package http

import (
	"consentcc/config"
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/fcp"
	"consentcc/internal/infra/client/minio"
	"consentcc/internal/infra/client/vault"
	"consentcc/internal/model"
	"consentcc/internal/service"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

type OrchestratorController struct {
	HealthChainService  service.HealthChainProvider
	OrchestratorService service.OrchestratorServiceProvider
	UserService         service.UserServiceProvider
	fc                  fabric.FabricClient
	minioClient         minio.IMinoIOClient
	vaultClient         vault.IVaultClient
	FcpClient           fcp.Client
	apiKey              string
	port                string
}

type IOrchestratorController interface {
	GetBundleDetail(ctx echo.Context) error
	RegisterUser(ctx echo.Context) error
	LoginUser(ctx echo.Context) error
	GetMe(ctx echo.Context) error
	GetBundlesGrantedToDoctorPaged(ctx echo.Context) error
	SearchDoctorByName(ctx echo.Context) error
	InitLedger(ctx echo.Context) error
	GetPatientByHash(ctx echo.Context) error
	ExternalGetBundleDetailFor(ctx echo.Context) error
}

func NewOrchestratorHTTPController(
	healthChainService service.HealthChainProvider,
	orchestratorService service.OrchestratorServiceProvider,
	userService service.UserServiceProvider,
	fc fabric.FabricClient,
	minioClient minio.IMinoIOClient,
	vaultClient vault.IVaultClient,
	FcpClient fcp.Client,
	cfg *config.Config) *OrchestratorController {
	// Add more endpoints later (list consents, bundles, audits)
	id, err := vaultClient.ReadHospitalID(context.Background())
	if err != nil {
		panic(err)
	}
	entity.SetHospitalID(id)
	controller := OrchestratorController{
		HealthChainService:  healthChainService,
		fc:                  fc,
		minioClient:         minioClient,
		vaultClient:         vaultClient,
		OrchestratorService: orchestratorService,
		UserService:         userService,
		FcpClient:           FcpClient,
		apiKey:              cfg.APIKey,
		port:                cfg.HTTPAddr,
	}
	return &controller

}

func (oc *OrchestratorController) GetBundleDetail(ctx echo.Context) error {
	// Implementation for getting bundle details
	//userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)
	id := ctx.Param("id")

	if id == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "missing bundle id"})
	}

	bundleData, err := oc.HealthChainService.ReadBundle(ctx.Request().Context(), id, nationalIdHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if bundleData == nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "bundle not found"})
	}

	if bundleData.URI != "" {
		data, err := oc.OrchestratorService.GetBundlesDetail(ctx.Request().Context(), []*entity.BundleFromFabric{bundleData}, nationalIdHash, role)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if len(data) == 0 {
			return ctx.JSON(http.StatusNotFound, map[string]string{"error": "not found any bundle detail"})
		}

		response := map[string]any{"ok": true, "bundle": bundleData, "data": data[0]}

		if bundleData.IsExternal != nil && !*bundleData.IsExternal {
			citizen, err := oc.UserService.GetUserByNationalIDHash(ctx.Request().Context(), bundleData.Owner)
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}

			if citizen != nil {
				response["user"] = citizen.User
			}
		}

		return ctx.JSON(http.StatusOK, response)
	}

	return nil
}

func (oc *OrchestratorController) RegisterUser(ctx echo.Context) error {
	// Implementation for registering a user
	var req entity.ReqRegisterUser
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user, err := oc.UserService.RegisterUser(ctx.Request().Context(), req.Role, req.CountryCode, req.NationalID, req.Name)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "user": user})
}

func (oc *OrchestratorController) LoginUser(ctx echo.Context) error {
	// Implementation for user login
	var req entity.ReqLoginUser
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	token, err := oc.UserService.LoginUser(ctx.Request().Context(), &req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "token": token})
}

func (oc *OrchestratorController) GetMe(ctx echo.Context) error {
	// Implementation for getting current user info
	userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	userInfo := map[string]string{
		"userId":         userId,
		"role":           role,
		"nationalIdHash": nationalIdHash,
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "info": userInfo})
}

func (o *OrchestratorController) GetBundlesGrantedToDoctorPaged(ctx echo.Context) error {
	nationalIdHash := ctx.Get("nationalIdHash").(string)
	role := ctx.Get("role").(string)

	if model.Role(role) != model.RolesConst.Doctor {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only doctor can get granted bundles"})
	}

	patientHash := ctx.QueryParam("patientHash")
	nextPageToken := ctx.QueryParam("nextPageToken")

	data, err := o.HealthChainService.GetBundlesGrantedToDoctorPaged(ctx.Request().Context(), patientHash, nationalIdHash, role, "10", nextPageToken)
	if err != nil {
		if err.Error() == "patient not found" {
			return ctx.JSON(http.StatusBadRequest, map[string]any{"error": err.Error(), "error_code": "USER_NOT_FOUND"})
		}
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	if len(data.Bundles) == 0 {
		return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": &entity.ListBundleUsersResponseWithNextToken{Bundles: nil, NextToken: data.NextToken}})
	}

	result, err := o.OrchestratorService.GetUsersInfoForBundles(ctx.Request().Context(), data.Bundles)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": &entity.ListBundleUsersResponseWithNextToken{Bundles: result, NextToken: data.NextToken}})
}

func (o *OrchestratorController) SearchDoctorByName(ctx echo.Context) error {
	doctorName := ctx.QueryParam("name")
	pageSize := ctx.QueryParam("pageSize")
	users, err := o.UserService.SearchDoctorByNameAndRole(ctx.Request().Context(), doctorName, string(model.RolesConst.Doctor), pageSize)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if users == nil || len(*users) == 0 {
		return ctx.JSON(http.StatusNotFound, map[string]any{"ok": true, "result": []entity.User{}})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": users})
}

func (oc *OrchestratorController) InitLedger(ctx echo.Context) error {
	var reqSchema = map[string]string{"BundleID": "bundle_id", "RequesterDoctorHash:": "doctor_hash", "RequesterPatientHash": "patient_hash"}
	bytes, _ := json.Marshal(reqSchema)

	hospitalid, err := oc.vaultClient.ReadHospitalID(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var respSchema = map[string]any{
		"ok":     "bool",
		"result": entity.Bundle{},
	}
	bytesResp, _ := json.Marshal(respSchema)

	var listFederationInterface = make([]fcp.FederationInterfaceDTO, 0)

	federationBundleDetailDTO := fcp.FederationInterfaceDTO{
		Name:    fcp.InterfaceGetBundleDetail,
		Version: "v1",
		Method:  http.MethodPost,
		Path:    "/api/v1/bundles/content",
		Auth: &fcp.AuthSchema{
			Type:   "BASIC",
			Header: "X-FCP-API-Key",
			Scheme: "ApiKey",
		},
		Encryption: &fcp.EncryptionSchema{
			RequestEncryption:  "NONE",
			ResponseEncryption: "RSA-OAEP-256",
			Alg:                "RSA",
		},
		Request:  bytes,
		Response: bytesResp,
		Type:     "API",
	}

	listFederationInterface = append(listFederationInterface, federationBundleDetailDTO)

	federationCheckWhitelistDTO := fcp.FederationInterfaceDTO{
		Name:    fcp.InterfaceCheckWhitelist,
		Version: "v1",
		Method:  http.MethodPost,
		Path:    "/api/v1/whitelist",
		Auth: &fcp.AuthSchema{
			Type:   "BASIC",
			Header: "X-FCP-API-Key",
			Scheme: "ApiKey",
		},
		Type: "Callback",
	}

	listFederationInterface = append(listFederationInterface, federationCheckWhitelistDTO)

	publicKey, _, err := oc.vaultClient.GetTransitPublicKeyForPayload(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	baseURL := fmt.Sprintf("http://localhost%s", oc.port)
	reqFcp := fcp.RegisterHospitalRequest{
		HospitalID:          hospitalid,
		Name:                "Orchestrator Hospital Cho Ray",
		BaseURL:             baseURL,
		BaseURLVersion:      "1",
		FederationInterface: listFederationInterface,
		PublicKey:           publicKey,
		KeyType:             "ed25519",
	}

	hods, err := oc.UserService.SearchHodByListNationalHash(ctx.Request().Context(), nil)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if hods != nil && len(*hods) > 0 {
		listID := make([]string, 0)
		for _, hod := range *hods {
			listID = append(listID, hod.NationalIDHash)
		}

		reqFcp.ListHODID = listID
	}

	resp, err := oc.FcpClient.RegisterHospital(ctx.Request().Context(), reqFcp)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "message": "ledger initialized successfully", "data": resp.Result})
}

func (oc *OrchestratorController) GetPatientByHash(ctx echo.Context) error {
	hash := ctx.QueryParam("hash")
	if hash == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Missing hash"})
	}

	cz, err := oc.UserService.GetUserByNationalIDHash(ctx.Request().Context(), hash)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if cz == nil || cz.User == nil || cz.User.Role != model.RolesConst.Patient {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"ok": true, "result": cz.User})
}

func (oc *OrchestratorController) ExternalGetBundleDetailFor(ctx echo.Context) error {
	type reqGetBundles struct {
		BundleID    string `json:"bundle_id"`
		DoctorHash  string `json:"doctor_hash"`
		PatientHash string `json:"patient_hash"`
	}
	log.Default().Println("GetListBundlesDetail started")
	var req reqGetBundles

	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	log.Default().Println("ReadListBundles")

	if req.DoctorHash == "" && req.PatientHash == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "unknown user"})
	}

	role := ""
	currentUserHash := ""
	if req.DoctorHash != "" {
		role = string(model.RolesConst.Doctor)
		currentUserHash = req.DoctorHash
	} else if req.PatientHash != "" {
		role = string(model.RolesConst.Patient)
		currentUserHash = req.PatientHash
	}
	bundleData, err := oc.HealthChainService.ReadBundle(ctx.Request().Context(), req.BundleID, currentUserHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if bundleData == nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "bundle not found"})
	}

	data, err := oc.OrchestratorService.GetBundlesDetail(ctx.Request().Context(), []*entity.BundleFromFabric{bundleData}, currentUserHash, role)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if len(data) == 0 {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "not found any bundle detail"})
	}

	var payload struct {
		OK     bool                     `json:"OK"`
		Bundle *entity.BundleFromFabric `json:"bundle"`
		User   *entity.User             `json:"user"`
	}
	payload.OK = true
	payload.Bundle = data[0]

	log.Default().Println("bundlesData", bundleData, *bundleData.IsExternal)
	if bundleData.IsExternal != nil && !*bundleData.IsExternal {
		citizen, err := oc.UserService.GetUserByNationalIDHash(ctx.Request().Context(), bundleData.Owner)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		log.Default().Println("GetUser", citizen)
		if citizen != nil {
			payload.User = citizen.User
		}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{"error": fmt.Errorf("marshal payload: %w", err).Error()})
	}

	log.Default().Println("SignWithTransitForPayload")
	sign, err := oc.vaultClient.SignWithTransitForPayload(ctx.Request().Context(), raw)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	log.Default().Println("Signed")

	return ctx.JSON(http.StatusOK, map[string]any{"signature": sign, "payload": base64.StdEncoding.EncodeToString(raw)})
}
