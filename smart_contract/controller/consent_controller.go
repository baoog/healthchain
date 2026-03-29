package controller

import (
	"consentcc/model"
	"consentcc/service"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"strconv"
	"strings"
	"time"
)

type ConsentContract struct {
	contractapi.Contract
	consentService service.ConsentServiceInterface
}

func NewConsentService(consentService service.ConsentServiceInterface) *ConsentContract {
	return &ConsentContract{
		consentService: consentService,
	}
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// CreateBundle(id, owner, typ, uri, sha256, size, archetypeId)
func (c *ConsentContract) CreateBundle(ctx contractapi.TransactionContextInterface,
	id, owner, typ, uri, sha256, size, archetypeId, signature, currentRole, currentUserHash string) (*model.Bundle, error) {
	hospital, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	b, err := c.consentService.CreateBundle(ctx, id, owner, typ, uri, sha256, size, archetypeId, signature, currentRole, currentUserHash, hospital)

	if err != nil {
		return nil, err
	}

	return b, nil
}

// GrantConsent(bundleId, doctorId, scope, purpose, expiresAt)
func (c *ConsentContract) GrantConsent(ctx contractapi.TransactionContextInterface,
	bundleId, currentId, currentRole, doctorId, purpose, expiresAt string) error {
	log.Default().Println("GrantConsent called with params:", bundleId, currentId, currentRole, doctorId, purpose, expiresAt)
	h, err := requireHospitalActive(ctx)
	if err != nil {
		return err
	}

	log.Println("GrantConsent", bundleId, currentId, currentRole, doctorId, purpose, expiresAt)
	if model.Role(currentRole) != model.RolesConst.Patient {
		return fmt.Errorf("only patient can grant consent")
	}

	err = c.consentService.GrantConsent(ctx, bundleId, currentId, currentRole, doctorId, purpose, expiresAt, h)
	if err != nil {
		return err
	}
	return nil
}

// RevokeConsent(bundleId, doctorId)
func (c *ConsentContract) RevokeConsent(ctx contractapi.TransactionContextInterface,
	currentUserId, currentRole, bundleId, doctorId string) error {
	h, err := requireHospitalActive(ctx)
	if err != nil {
		return err
	}

	if model.Role(currentRole) != model.RolesConst.Patient {
		return fmt.Errorf("only patient can grant consent")
	}

	return c.consentService.RevokeConsent(ctx, currentUserId, currentRole, bundleId, doctorId, h)
}

func (c *ConsentContract) ReadBundle(ctx contractapi.TransactionContextInterface, id, userid, role string) (*model.Bundle, error) {
	h, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	b, err := c.consentService.ReadBundle(ctx, id, userid, role, h)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (c *ConsentContract) ReadListBundles(ctx contractapi.TransactionContextInterface, listids, userid, role string) ([]*model.Bundle, error) {
	h, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	listBundleID := strings.Split(listids, ",")
	listBundles := make([]*model.Bundle, 0)
	for _, id := range listBundleID {
		b, err := c.consentService.ReadBundle(ctx, id, userid, role, h)
		if err != nil {
			log.Default().Println(err.Error())
			continue
		}

		listBundles = append(listBundles, b)
	}

	return listBundles, nil
}

func (c *ConsentContract) ReadConsent(ctx contractapi.TransactionContextInterface, bundleId, currentUserId, currentRole string) ([]model.Consent, error) {
	h, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if model.Role(currentRole) != model.RolesConst.Patient && model.Role(currentRole) != model.RolesConst.Doctor {
		return nil, fmt.Errorf("access denied")
	}

	consents, err := c.consentService.ReadConsent(ctx, bundleId, currentUserId, currentRole, h)
	if err != nil {
		return nil, err
	}
	return consents, nil
}

func (c *ConsentContract) GetBundlesByOwner(ctx contractapi.TransactionContextInterface,
	owner string) ([]*model.Bundle, error) {
	_, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	bundles, err := c.consentService.GetBundlesByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}

	return bundles, nil
}

// ListConsentsByOwner(ownerId)
// Returns all *active* consents for bundles owned by ownerId.
func (c *ConsentContract) ListConsentsByOwner(ctx contractapi.TransactionContextInterface,
	ownerId, currentRole string) ([]model.ConsentWithBundle, error) {

	h, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if model.Role(currentRole) != model.RolesConst.Patient {
		return nil, fmt.Errorf("only patient can list consents by owner")
	}
	result, err := c.consentService.ListConsentsByOwner(ctx, ownerId, currentRole, h)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ConsentContract) GetBundlesGrantedToDoctorPaged(
	ctx contractapi.TransactionContextInterface,
	ownerHash string,
	doctorHash string,
	currentRole string,
	pageSize string,
	bookmark string,
) (*model.ListBundleResponseWithNextToken, error) {
	_, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if model.Role(currentRole) != model.RolesConst.Doctor {
		return nil, fmt.Errorf("only doctor can query granted bundles")
	}

	return c.consentService.GetBundlesGrantedToDoctorPaged(ctx, ownerHash, doctorHash, pageSize, bookmark)
}

// ---------- Break-Glass (Emergency) ----------

// RequestBreakGlass(ownerHash, doctorHash, doctorRole, justification, expiresAt, allowedTypesCSV)
// doctorRole must be DOCTOR
func (c *ConsentContract) RequestBreakGlass(ctx contractapi.TransactionContextInterface,
	ownerHash, doctorHash, doctorRole, justification, expiresAt string, scopes string, rseq string) (*model.BreakGlassRequest, error) {

	hospital, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	log.Println("RequestBreakGlass", ownerHash, doctorHash, doctorRole, justification, expiresAt, scopes)
	if model.Role(doctorRole) != model.RolesConst.Doctor {
		log.Println("doctorRole invalid:", doctorRole)
		return nil, fmt.Errorf("only doctor can request break-glass")
	}
	if justification == "" {
		log.Println("justification is empty")
		return nil, fmt.Errorf("justification is required")
	}
	if expiresAt == "" {
		log.Println("expiresAt is empty")
		return nil, fmt.Errorf("expiresAt is required")
	}

	bg, err := c.consentService.RequestBreakGlass(ctx, ownerHash, doctorHash, justification, expiresAt, scopes, hospital.HospitalID, rseq)
	if err != nil {
		log.Println("RequestBreakGlass error:", err)
		return nil, err
	}

	return bg, nil
}

func (c *ConsentContract) ListBreakGlassByDoctor(
	ctx contractapi.TransactionContextInterface,
	doctorHash string,
	role string,
	pageSize string,
	bookmark string,
) (*model.ListBreakGlassRequestResponseWithNextToken, error) {
	_, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if model.Role(role) != model.RolesConst.Doctor {
		return nil, fmt.Errorf("only doctor can list break-glass requests")
	}

	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return nil, err
	}

	if pageSizeInt <= 0 || pageSizeInt > 100 {
		return nil, fmt.Errorf("pageSize must be between 1 and 100")
	}

	result, err := c.consentService.ListBreakGlassByDoctor(ctx, doctorHash, int32(pageSizeInt), bookmark)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ApproveBreakGlass(ownerHash, doctorHash, hodHash, hodRole, decision)
// hodRole must be HOD. decision ∈ {APPROVE, REJECT}
func (c *ConsentContract) ApproveBreakGlass(ctx contractapi.TransactionContextInterface,
	requestID, hodHash, decision string) error {

	hospital, err := requireHospitalActive(ctx)
	if err != nil {
		return err
	}

	if strings.HasPrefix("BG_", requestID) == false && len(strings.Split(requestID, "_")) < 4 {
		return fmt.Errorf("invalid break-glass request ID")
	}
	err = c.consentService.ApproveBreakGlass(ctx, requestID, hodHash, decision, hospital.HospitalID)
	if err != nil {
		return err
	}

	return nil
}

// GetEmergencyBundlesByOwner(ownerHash, doctorHash, doctorRole)
// doctorRole must be DOCTOR
func (c *ConsentContract) GetEmergencyBundlesFromDoctor(ctx contractapi.TransactionContextInterface,
	requestID, doctorHash, doctorRole string) ([]*model.Bundle, error) {
	_, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if model.Role(doctorRole) != model.RolesConst.Doctor && model.Role(doctorRole) != model.RolesConst.HoD {
		return nil, fmt.Errorf("only doctor can use emergency read")
	}

	if strings.HasPrefix("BG_", requestID) == false && len(strings.Split(requestID, "_")) < 4 {
		return nil, fmt.Errorf("invalid break-glass request ID")
	}

	out, err := c.consentService.GetEmergencyBundlesFromDoctor(ctx, requestID, doctorHash, doctorRole)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *ConsentContract) GetBreakGlassRequestByRequestID(
	ctx contractapi.TransactionContextInterface,
	requestID string,
	currentUserHash string,
	currentUserRole string,
) (*model.BreakGlassRequest, error) {
	_, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix("BG_", requestID) == false && len(strings.Split(requestID, "_")) < 4 {
		return nil, fmt.Errorf("invalid break-glass request ID")
	}

	if model.Role(currentUserRole) != model.RolesConst.Doctor && model.Role(currentUserRole) != model.RolesConst.HoD {
		return nil, fmt.Errorf("only doctor or HOD can get break-glass request")
	}

	bg, err := c.consentService.GetBreakGlassRequestByRequestID(ctx, requestID, currentUserHash, currentUserRole)
	if err != nil {
		return nil, err
	}

	return bg, nil
}

func (c *ConsentContract) GetPendingBreakGlassRequests(
	ctx contractapi.TransactionContextInterface,
) ([]*model.BreakGlassRequest, error) {

	hospital, err := requireHospitalActive(ctx)
	if err != nil {
		return nil, err
	}

	result, err := c.consentService.GetPendingBreakGlassRequests(ctx, hospital.HospitalID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *ConsentContract) RegisterHospital(
	ctx contractapi.TransactionContextInterface,
	hospitalID string,
	name string,
	createdAt string,
	listIDofHod string,
) (*model.Hospital, error) {
	if err := onlyNetworkAdmin(ctx); err != nil {
		return nil, err
	}

	if hospitalID == "" || name == "" {
		return nil, fmt.Errorf("hospitalID and name are required")
	}

	arrayIDHod := strings.Split(listIDofHod, ",")
	if len(arrayIDHod) == 0 {
		return nil, fmt.Errorf("array id HOD is required")
	}

	h, err := c.consentService.RegisterHospital(ctx, hospitalID, name, createdAt, arrayIDHod)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func assertHospitalActive(
	ctx contractapi.TransactionContextInterface,
	hospitalID string,
) (*model.Hospital, error) {

	key := fmt.Sprintf(model.KeyHospital, hospitalID)
	val, err := ctx.GetStub().GetState(key)
	if err != nil || len(val) == 0 {
		return nil, fmt.Errorf("hospital not registered")
	}

	var h model.Hospital
	_ = json.Unmarshal(val, &h)

	if h.Status != model.HospitalActive {
		return nil, fmt.Errorf("hospital %s is %s", hospitalID, h.Status)
	}
	return &h, nil
}

func onlyNetworkAdmin(ctx contractapi.TransactionContextInterface) error {
	cid := ctx.GetClientIdentity()
	role, ok, err := cid.GetAttributeValue("role")
	if err != nil {
		return err
	} else if !ok || role != "fcp" {
		return fmt.Errorf("only network admin can perform this action")
	}
	return nil
}

func requireHospitalActive(
	ctx contractapi.TransactionContextInterface,
) (*model.Hospital, error) {
	cid := ctx.GetClientIdentity()
	hospitalID, ok, err := cid.GetAttributeValue("hospital_id")
	if err != nil {
		return nil, err
	} else if !ok || hospitalID == "" {
		return nil, fmt.Errorf("hospital_id attribute missing in identity")
	}

	return assertHospitalActive(ctx, hospitalID)
}
