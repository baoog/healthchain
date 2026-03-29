package http

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/server"
	"consentcc/internal/model"
	"consentcc/internal/service"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strconv"
	"time"
)

type HealthChainController struct {
	consent             service.HealthChainProvider
	fc                  fabric.FabricClient
	orchestratorService service.OrchestratorServiceProvider
}

type IHealthChainController interface {
	Get(ctx echo.Context) error
	EvaluateFabric(ctx echo.Context) error
	CreateBundle(ctx echo.Context) error
	GrantConsent(ctx echo.Context) error
	RevokeConsent(ctx echo.Context) error
	//listAudits(ctx echo.Context) error
	ReadBundle(ctx echo.Context) error
	ReadConsent(ctx echo.Context) error
	GetBundlesByOwner(ctx echo.Context) error
	RequestEmergencyAccess(ctx echo.Context) error
	GetMyEmergencyAccessRequests(ctx echo.Context) error
	ApproveEmergencyAccess(ctx echo.Context) error
	GetBreakGlassRequestByRequestID(ctx echo.Context) error
	GetEmergencyBundlesForDoctor(ctx echo.Context) error
	GetEmergencyAccessRequestsPending(ctx echo.Context) error
	GetBundlesGrantedToDoctorPaged(ctx echo.Context) error
	ListMyConsents(ctx echo.Context) error
}

func NewHTTPController(
	s server.EchoServer,
	consent service.HealthChainProvider,
	fc fabric.FabricClient,
	orchestratorService service.OrchestratorServiceProvider) *HealthChainController {
	// Add more endpoints later (list consents, bundles, audits)

	controller := HealthChainController{consent: consent, fc: fc, orchestratorService: orchestratorService}
	return &controller

}

func (hc *HealthChainController) Get(ctx echo.Context) error {
	hc.fc.Contract().ContractName()
	return ctx.String(http.StatusOK, "HealthChainController is running")
}

func (c *HealthChainController) EvaluateFabric(ctx echo.Context) error {
	type EvaluateReq struct {
		Fcn  string   `json:"fcn"`
		Args []string `json:"args"`
	}

	req := new(EvaluateReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	log.Default().Println("EvaluateFabric called with Fcn:", req.Fcn, "Args:", req.Args)
	// Calls a read-only chaincode function
	res, err := c.fc.Contract().SubmitTransaction(req.Fcn, req.Args...)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": string(res)})
}

func (c *HealthChainController) listAudits(ctx echo.Context) error {
	//limit, offset := parsePage(ctx)
	//items, err := c.audit.List(ctx.Request().Context(), limit, offset)
	//if err != nil {
	//	return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	//}
	return ctx.JSON(http.StatusOK, nil)
}

func parsePage(c echo.Context) (int, int) {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (c *HealthChainController) CreateBundle(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)
	role := ctx.Get("role").(string)

	req := new(entity.CreateBundleReq)
	//bytes, _ := json.Marshal(bytes2.NewReader(ctx.Request().))
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	seedHash := nationalIdHash
	if model.Role(role) == model.RolesConst.Patient {
		req.Owner = nationalIdHash
	} else if model.Role(role) == model.RolesConst.Doctor {
		if req.Owner == "" {
			return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "missing owner hash"})
		}
		seedHash = req.Owner
	}

	bundle, err := c.consent.CreateBundle(ctx.Request().Context(), req, seedHash, role, nationalIdHash)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": bundle})
}

func (c *HealthChainController) GrantConsent(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	req := new(entity.GrantConsentReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err := c.consent.GrantConsent(ctx.Request().Context(), req, nationalIdHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (c *HealthChainController) RevokeConsent(ctx echo.Context) error {
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	req := new(entity.RevokeConsentReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	res, err := c.consent.RevokeConsent(
		ctx.Request().Context(),
		nationalIdHash, role, req.BundleID, req.DoctorID,
	)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": res})
}

func (c *HealthChainController) ReadBundle(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	id := ctx.Param("id")
	res, err := c.consent.ReadBundle(ctx.Request().Context(), id, nationalIdHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": res})
}

func (c *HealthChainController) ReadConsent(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	b := ctx.Param("bundleId")
	res, err := c.fc.Contract().EvaluateTransaction("ReadConsent", b, nationalIdHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var dataResponse []entity.Consent
	err = json.Unmarshal(res, &dataResponse)
	if err != nil {
		fmt.Println(err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": dataResponse})
}

func (c *HealthChainController) GetBundlesByOwner(ctx echo.Context) error {
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	res, err := c.fc.Contract().EvaluateTransaction("GetBundlesByOwner", nationalIdHash)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var bundles []*entity.BundleFromFabric
	if err := json.Unmarshal(res, &bundles); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for _, b := range bundles {
		if b.HospitalId != "" && b.HospitalId != entity.GetHospitalID() {
			True := true
			b.IsExternal = &True
		} else if b.HospitalId != "" && b.HospitalId == entity.GetHospitalID() {
			False := false
			b.IsExternal = &False
		}
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": bundles})
}

func (c *HealthChainController) ListMyConsents(ctx echo.Context) error {
	roleVal := ctx.Get("role")
	role, _ := roleVal.(string)

	if model.Role(role) != model.RolesConst.Patient {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only patient can list consents on their data"})
	}

	nationalIdHash, _ := ctx.Get("nationalIdHash").(string)
	if nationalIdHash == "" {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "userId not in context"})
	}

	// Call Fabric: ListConsentsByOwner(ownerId=userId)
	items, err := c.consent.ListMyConsent(ctx.Request().Context(), nationalIdHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"ok":      true,
		"ownerId": nationalIdHash,
		"result":  items,
	})
}

func (c *HealthChainController) GetBundlesGrantedToDoctorPaged(ctx echo.Context) error {
	nationalIdHash := ctx.Get("nationalIdHash").(string)
	role := ctx.Get("role").(string)

	if model.Role(role) != model.RolesConst.Doctor {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only doctor can get granted bundles"})
	}

	patientHash := ctx.QueryParam("patientHash")
	nextPageToken := ctx.QueryParam("nextPageToken")

	data, err := c.consent.GetBundlesGrantedToDoctorPaged(ctx.Request().Context(), patientHash, nationalIdHash, role, "100", nextPageToken)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": data})

}

func (c *HealthChainController) RequestEmergencyAccess(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	nationalIdHash := ctx.Get("nationalIdHash").(string)
	role := ctx.Get("role").(string)

	req := new(entity.EmergencyAccessRequest)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.ExpiresInMinutes == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "expiresInMinutes is required"})
	}
	_, err := time.ParseDuration(req.ExpiresInMinutes)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiresInMinutes format"})
	}

	if len(req.Types) == 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "at least one type is required"})
	}

	res, err := c.consent.RequestEmergencyAccess(
		ctx.Request().Context(),
		req,
		nationalIdHash, // doctor's pseudonymous ID
		role,           // should be "DOCTOR"
	)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": res})
}

func (c *HealthChainController) ApproveEmergencyAccess(ctx echo.Context) error {
	nationalIdHash := ctx.Get("nationalIdHash").(string)

	req := new(entity.EmergencyAccessDecisionReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	status, ok := model.MapEmergencyAccessStatus[req.Decision]
	if !ok {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid decision value"})
	}

	err := c.consent.ApproveEmergencyAccess(
		ctx.Request().Context(),
		req.RequestID,
		nationalIdHash,
		status)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": "ok"})
}

func (c *HealthChainController) GetMyEmergencyAccessRequests(ctx echo.Context) error {
	role := ctx.Get("role").(string)

	if model.Role(role) != model.RolesConst.Doctor {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only doctor can get granted bundles"})
	}

	nextPageToken := ctx.QueryParam("nextPageToken")
	nationalIdHash, ok := ctx.Get("nationalIdHash").(string)
	if !ok {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "missing nationalIdHash in token"})
	}

	res, err := c.consent.GetMyBreakGlassRequests(ctx.Request().Context(), nationalIdHash, role, "100", nextPageToken)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": res})
}

func (c *HealthChainController) GetBreakGlassRequestByRequestID(ctx echo.Context) error {
	//userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)

	userHash, ok := ctx.Get("nationalIdHash").(string)
	if !ok {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "missing nationalIdHash in token"})
	}

	requestID := ctx.Param("requestID")
	res, err := c.fc.Contract().SubmitTransaction("GetBreakGlassRequestByRequestID", requestID, userHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var emergencyAccess entity.BreakGlassRequestContract
	err = json.Unmarshal(res, &emergencyAccess)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid chaincode response"})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": emergencyAccess})
}

func (c *HealthChainController) GetEmergencyBundlesForDoctor(ctx echo.Context) error {
	role := ctx.Get("role").(string)
	requestID := ctx.Param("requestId")
	userHash := ctx.Get("nationalIdHash").(string)

	if model.Role(role) != model.RolesConst.Doctor && model.Role(role) != model.RolesConst.HoD {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only doctor can get emergency bundles"})
	}

	bundles, err := c.consent.GetEmergencyBundlesForDoctor(ctx.Request().Context(), requestID, userHash, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	log.Default().Println(bundles)

	bundles, err = c.orchestratorService.GetBundlesDetail(ctx.Request().Context(), bundles, userHash, role)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": bundles})
}

func (c *HealthChainController) GetEmergencyAccessRequestsPending(ctx echo.Context) error {
	role := ctx.Get("role").(string)

	if model.Role(role) != model.RolesConst.HoD {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only HOD can get pending emergency access requests"})
	}

	res, err := c.consent.GetEmergencyAccessRequestsPending(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": res})
}
