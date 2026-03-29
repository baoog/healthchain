package service

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/adapter/repository"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/minio"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/infra/client/vault"
	"consentcc/internal/model"
	"consentcc/utils"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"strings"
	"time"
)

type HealthChainService struct {
	db       *gorm.DB
	fc       fabric.FabricClient
	minioC   minio.IMinoIOClient
	vaultC   vault.IVaultClient
	userRepo repository.UserRepoProvider
}

type HealthChainProvider interface {
	CreateBundle(ctx context.Context, req *entity.CreateBundleReq, ownerHash, currentRole, currentUserHash string) (*entity.Bundle, error)
	ReadBundle(ctx context.Context, bundleId, currentUserId, currentRole string) (*entity.BundleFromFabric, error)
	//GetBundleByID(ctx context.Context, bundleId string) (*entity.Bundle, error)
	//GetBundlesByOwner(ctx context.Context, ownerId string, limit, offset int) ([]*entity.Bundle, error)
	GrantConsent(ctx context.Context, req *entity.GrantConsentReq, currentUserID, currentRole string) error
	RevokeConsent(ctx context.Context, currentUserId, currentRole, bundleId, doctorId string) (interface{}, error)
	GetBundlesGrantedToDoctorPaged(ctx context.Context, patientNationalHash, doctorHash, role, pageSize, nextPageToken string) (*entity.ListBundleResponseWithNextToken, error)
	ListMyConsent(ctx context.Context, fabricID string, role string) ([]*entity.ConsentWithBundle, error)
	//GetConsent(ctx context.Context, bundleId, doctorId string) (*entity.Consent, error)
	RequestEmergencyAccess(ctx context.Context, req *entity.EmergencyAccessRequest, currentUserId, currentRole string) (*entity.EmergencyAccess, error)
	ApproveEmergencyAccess(ctx context.Context, requestID, hod string, decision model.EmergencyAccessStatus) error
	GetMyBreakGlassRequests(ctx context.Context, doctorHash, role, limit, nextPageToken string) (*entity.ListBreakGlassRequestResponseWithNextToken, error)
	GetEmergencyAccessRequestsPending(ctx context.Context) ([]*entity.BreakGlassRequestContract, error)
	GetEmergencyBundlesForDoctor(ctx context.Context, requestID, currentUserID, role string) ([]*entity.BundleFromFabric, error)
	ReadListBundles(ctx context.Context, currentUseNationalHash, currentRole string, listBundleIDs []string) ([]*entity.BundleFromFabric, error)
}

func NewHealthChainService(
	dbClient *postgres.Client,
	fabricC fabric.FabricClient,
	minioC minio.IMinoIOClient,
	vaultC vault.IVaultClient,
	userRepo repository.UserRepoProvider,
) *HealthChainService {
	return &HealthChainService{
		db:       dbClient.DB,
		fc:       fabricC,
		minioC:   minioC,
		vaultC:   vaultC,
		userRepo: userRepo,
	}
}

type EventPayload struct {
	Func string   `json:"func"`
	Args []string `json:"args"`
}

func generateBundleID(ownerId string) string {
	// Simple example; you can make it nicer (uuid, timestamp, etc.)
	return fmt.Sprintf("bndl-%s-%d", ownerId, time.Now().UnixNano())
}

func (s *HealthChainService) CreateBundle(ctx context.Context, req *entity.CreateBundleReq, ownerHash, currentRole, currentUserHash string) (*entity.Bundle, error) {
	cz, err := s.userRepo.SearchByNationalIDHash(ctx, ownerHash)
	if err != nil {
		return nil, err
	}

	if cz == nil || cz.User == nil {
		return nil, fmt.Errorf("user not found")
	}

	bundleId := generateBundleID(cz.User.ID)
	req.ID = bundleId

	dataKey, err := utils.GenerateDataKey()
	if err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(req.Data)
	if err != nil {
		return nil, err
	}

	cipher, err := utils.EncryptBundle(payloadBytes, dataKey)
	if err != nil {
		return nil, err
	}
	req.Size = fmt.Sprintf("%d", cipher.Size)
	req.SHA256 = cipher.SHA256Hex

	reader := cipher.BuildReader()
	uri, err := s.minioC.PutBundle(ctx, ownerHash, bundleId, reader, cipher.Size)
	if err != nil {
		return nil, err
	}

	hashBytes, err := hex.DecodeString(cipher.SHA256Hex) // 32 bytes
	signature, err := s.vaultC.SignWithTransitForBundle(ctx, hashBytes)
	if err != nil {
		return nil, fmt.Errorf("sign bundle hash: %w", err)
	}

	// Call chaincode to create bundle record
	res, err := s.fc.Contract().SubmitTransaction(
		"CreateBundle",
		bundleId,
		req.Owner,
		string(req.Type),
		uri, req.SHA256,
		req.Size,
		req.ArchetypeID,
		signature,
		currentRole,
		currentUserHash,
	)
	if err != nil {
		return nil, err
	}

	// Store data key in Vault
	err = s.vaultC.StoreBundleKey(ctx, bundleId, "ehr/keys", dataKey)
	if err != nil {
		return nil, err
	}

	var bundle entity.Bundle
	err = json.Unmarshal(res, &bundle)
	if err != nil {
		return nil, err
	}

	return &bundle, nil
}

func (s *HealthChainService) GrantConsent(ctx context.Context, req *entity.GrantConsentReq, currentUserFabricIDHashed, currentRole string) error {
	doctor, err := s.userRepo.SearchByNationalIDHash(ctx, req.DoctorID)
	if err != nil {
		return err
	}
	if doctor == nil || doctor.User == nil {
		return fmt.Errorf("doctor not found")
	}

	for _, bundleID := range req.BundleIDs {
		_, err = s.fc.Contract().SubmitTransaction(
			"GrantConsent",
			bundleID,
			currentUserFabricIDHashed,
			currentRole,
			doctor.NationalIDHash,
			req.Purpose,
			req.ExpiresAt,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *HealthChainService) RevokeConsent(ctx context.Context, currentFabricHash, currentRole, bundleId, doctorId string) (interface{}, error) {
	doctor, err := s.userRepo.SearchByNationalIDHash(ctx, doctorId)
	if err != nil {
		return nil, err
	}

	if doctor == nil || doctor.User == nil {
		return nil, fmt.Errorf("doctor not found")
	}

	res, err := s.fc.Contract().SubmitTransaction(
		"RevokeConsent",
		currentFabricHash,
		currentRole,
		bundleId,
		doctor.NationalIDHash,
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *HealthChainService) ReadBundle(ctx context.Context, bundleId, currentFabricHash, currentRole string) (*entity.BundleFromFabric, error) {
	res, err := s.fc.Contract().EvaluateTransaction(
		"ReadBundle",
		bundleId,
		currentFabricHash,
		currentRole)
	if err != nil {
		return nil, err
	}

	var dataResponse entity.BundleFromFabric
	_ = json.Unmarshal(res, &dataResponse)

	if dataResponse.HospitalId != "" && dataResponse.HospitalId != entity.GetHospitalID() {
		True := true
		dataResponse.IsExternal = &True
	} else if dataResponse.HospitalId != "" && dataResponse.HospitalId == entity.GetHospitalID() {
		False := false
		dataResponse.IsExternal = &False
	}
	return &dataResponse, nil
}

func (s *HealthChainService) GetBundlesGrantedToDoctorPaged(ctx context.Context, patientNationalHash, doctorHash, role, pageSize, nextPageToken string) (*entity.ListBundleResponseWithNextToken, error) {
	fmt.Println("patientNationalHash", patientNationalHash)
	patient, err := s.userRepo.SearchByNationalIDHash(ctx, patientNationalHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("patient not found")
		}
		return nil, err
	}
	if patient == nil || patient.User == nil {
		patient = &entity.CitizenIdentity{
			NationalIDHash: patientNationalHash,
			User:           &entity.User{},
		}
	}

	res, err := s.fc.Contract().EvaluateTransaction(
		"GetBundlesGrantedToDoctorPaged",
		patient.NationalIDHash,
		doctorHash,
		role,
		pageSize,
		nextPageToken)
	if err != nil {
		return nil, err
	}

	var resData entity.ListBundleResponseWithNextToken
	if err := json.Unmarshal(res, &resData); err != nil {
		return nil, err
	}

	for _, b := range resData.Bundles {
		if b.HospitalId != "" && b.HospitalId != entity.GetHospitalID() {
			True := true
			b.IsExternal = &True
		} else if b.HospitalId != "" && b.HospitalId == entity.GetHospitalID() {
			False := false
			b.IsExternal = &False
		}
	}

	return &resData, nil
}

func (s *HealthChainService) ListMyConsent(ctx context.Context, nationalHash string, role string) ([]*entity.ConsentWithBundle, error) {
	res, err := s.fc.Contract().EvaluateTransaction("ListConsentsByOwner", nationalHash, role)
	if err != nil {
		return nil, err
	}

	var items []*entity.ConsentWithBundle // mirror of model.ConsentWithBundle
	if err := json.Unmarshal(res, &items); err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return items, nil
	}

	var (
		listDoctorID []string
		mapDupCheck  = make(map[string]struct{})
	)
	for _, c := range items {
		if _, ok := mapDupCheck[c.Consent.GranteeHash]; !ok {
			// Grantee Hash == nationalHash of doctor
			mapDupCheck[c.Consent.GranteeHash] = struct{}{}
			listDoctorID = append(listDoctorID, c.Consent.GranteeHash)
		}
	}

	_, mapUserInfo, err := s.userRepo.LoadUsersByListNationalHash(ctx, listDoctorID)
	if err != nil {
		return nil, err
	}

	for _, c := range items {
		if user, ok := mapUserInfo[c.Consent.GranteeHash]; ok {
			c.Grantee = &entity.Grantee{
				ID:             user.UserID,
				Name:           user.User.Name,
				Role:           user.User.Role,
				NationalIDHash: user.NationalIDHash,
			}
		}
	}

	return items, nil
}

func (s *HealthChainService) RequestEmergencyAccess(ctx context.Context, req *entity.EmergencyAccessRequest, currentFabricHashed, currentRole string) (*entity.EmergencyAccess, error) {
	patient, err := s.userRepo.SearchByNationalIDHash(ctx, req.PatientHash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	isExternal := false
	if patient == nil || patient.User == nil {
		isExternal = true
		patient = &entity.CitizenIdentity{NationalIDHash: req.PatientHash}
	}

	scopeStr := strings.Join(req.Types, ",")

	args := []string{
		patient.NationalIDHash,
		currentFabricHashed,
		currentRole,
		req.Justification,
		req.ExpiresInMinutes,
		scopeStr,
		uuid.NewString(),
	}

	log.Default().Println("RequestEmergencyAccess args: ", args, len(args))
	res, err := s.fc.Contract().SubmitTransaction(
		"RequestBreakGlass", args...,
	)
	if err != nil {
		return nil, err
	}

	var emergencyAccess entity.EmergencyAccess
	err = json.Unmarshal(res, &emergencyAccess)
	if err != nil {
		return nil, err
	}

	emergencyAccess.IsExternal = &isExternal

	return &emergencyAccess, nil
}

func (s *HealthChainService) ApproveEmergencyAccess(ctx context.Context, requestID, hodHash string, decision model.EmergencyAccessStatus) error {
	_, err := s.fc.Contract().SubmitTransaction(
		"ApproveBreakGlass",
		requestID,
		hodHash,
		string(decision),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *HealthChainService) GetMyBreakGlassRequests(ctx context.Context, doctorHash, role, limit, nextPageToken string) (*entity.ListBreakGlassRequestResponseWithNextToken, error) {
	res, err := s.fc.Contract().EvaluateTransaction(
		"ListBreakGlassByDoctor",
		doctorHash,
		role,
		limit,
		nextPageToken,
	)
	if err != nil {
		return nil, err
	}

	var requests *entity.ListBreakGlassRequestResponseWithNextToken
	err = json.Unmarshal(res, &requests)
	if err != nil {
		return nil, err
	}

	var (
		seen      = make(map[string]struct{})
		listOwner []string
	)
	for _, req := range requests.Requests {
		if !req.IsExternal {
			if _, ok := seen[req.OwnerHash]; !ok {
				listOwner = append(listOwner, req.OwnerHash)
				seen[req.OwnerHash] = struct{}{}
			}
		}
	}

	_, mapUser, err := s.userRepo.LoadUsersByListNationalHash(ctx, listOwner)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	for _, req := range requests.Requests {
		if citizen, ok := mapUser[req.OwnerHash]; ok {
			req.Owner = citizen.User
		} else {
			req.Owner = &entity.User{Name: "Unknown"}
		}
	}

	return requests, nil
}

func (s *HealthChainService) GetEmergencyAccessRequestsPending(ctx context.Context) ([]*entity.BreakGlassRequestContract, error) {
	res, err := s.fc.Contract().SubmitTransaction("GetPendingBreakGlassRequests")
	if err != nil {
		return nil, err
	}

	if res == nil {
		return []*entity.BreakGlassRequestContract{}, nil
	}

	var requests []*entity.BreakGlassRequestContract
	if err = json.Unmarshal(res, &requests); err != nil {
		return nil, err
	}

	if len(requests) == 0 {
		return requests, nil
	}

	var (
		mapCheckdupOwner = make(map[string]struct{})
		listOwner        []string
	)

	for _, b := range requests {
		if _, ok := mapCheckdupOwner[b.OwnerHash]; !ok {
			mapCheckdupOwner[b.OwnerHash] = struct{}{}
			listOwner = append(listOwner, b.OwnerHash)
		}
	}

	_, mapUsers, err := s.userRepo.LoadUsersByListNationalHash(ctx, listOwner)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	for _, r := range requests {
		if user, ok := mapUsers[r.OwnerHash]; ok {
			r.Owner = user.User
		} else {
			r.Owner = &entity.User{
				Name: "Unknown", // External patient
			}
		}
	}

	return requests, nil
}

func (s *HealthChainService) GetEmergencyBundlesForDoctor(ctx context.Context, requestID, currentUserHash, role string) ([]*entity.BundleFromFabric, error) {
	res, err := s.fc.Contract().EvaluateTransaction("GetEmergencyBundlesFromDoctor", requestID, currentUserHash, role)
	if err != nil {
		return nil, err
	}

	var bundles []*entity.BundleFromFabric
	if err := json.Unmarshal(res, &bundles); err != nil {
		log.Default().Println("Unmarshal", err.Error())
		return nil, err
	}

	hospitalID, err := s.vaultC.ReadHospitalID(ctx)
	if err != nil {
		return nil, err
	}

	for _, bundle := range bundles {
		if bundle.HospitalId != hospitalID {
			True := true
			bundle.IsExternal = &True
		} else {
			False := false
			bundle.IsExternal = &False
		}
	}

	return bundles, nil
}

func (s *HealthChainService) ReadListBundles(ctx context.Context, currentUseNationalHash, currentRole string, listBundleIDs []string) ([]*entity.BundleFromFabric, error) {
	ids := strings.Join(listBundleIDs, ",")
	res, err := s.fc.Contract().EvaluateTransaction("ReadListBundles", ids, currentUseNationalHash, currentRole)
	if err != nil {
		return nil, err
	}

	var bundles []*entity.BundleFromFabric
	if err := json.Unmarshal(res, &bundles); err != nil {
		log.Default().Println("Unmarshal", err.Error())
		return nil, err
	}

	hospitalId := entity.GetHospitalID()

	for _, bundle := range bundles {
		log.Default().Println(bundle.ID, bundle.HospitalId, hospitalId)
		if bundle.HospitalId != hospitalId {
			True := true
			bundle.IsExternal = &True
		} else {
			False := false
			bundle.IsExternal = &False
		}
		//log.Default().Println("CHEKCED", *bundle.IsExternal, bundle.ID, bundle.HospitalId, hospitalID)
	}

	return bundles, nil
}
