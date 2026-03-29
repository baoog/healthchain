package service

import (
	"consentcc/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	keyBreakGlass      = "BG_%s_%s_%v" // BG_<ownerHash>_<doctorHash>
	keyBundle          = "BNDL_%s"
	keyConsent         = "CONS_%s_%s"
	indexPatientBundle = "patient~bundle"
	indexOwnerConsent  = "owner~bundle~user" // composite key: ownerId, bundleId, userId
	indexBGStatus      = "BG_STATUS"
	indexBGDoctor      = "BG_DOCTOR"
	indexBGOwner       = "BG_OWNER"
	indexBGHospital    = "BG_HOSPITAL"
	indexBGBundle      = "BG_BUNDLE"
	indexHoD           = "HOD_HPT"
)

type (
	ConsentService struct{}

	ConsentServiceInterface interface {
		CreateBundle(ctx contractapi.TransactionContextInterface, id, owner, typ, uri, sha256, size, archetypeId, signature, currentRole, currentUserHash string, hosp *model.Hospital) (*model.Bundle, error)
		GrantConsent(ctx contractapi.TransactionContextInterface,
			bundleId, currentId, currentRole, doctorId, purpose, expiresAt string, h *model.Hospital) error
		ReadConsent(ctx contractapi.TransactionContextInterface, bundleId, currentUserId, currentRole string, h *model.Hospital) ([]model.Consent, error)
		RevokeConsent(ctx contractapi.TransactionContextInterface,
			currentUserId, currentRole, bundleId, doctorId string, h *model.Hospital) error
		ReadBundle(ctx contractapi.TransactionContextInterface, id, userid, role string, h *model.Hospital) (*model.Bundle, error)
		ListConsentsByOwner(ctx contractapi.TransactionContextInterface,
			ownerId, currentRole string, h *model.Hospital) ([]model.ConsentWithBundle, error)
		checkConsent(ctx contractapi.TransactionContextInterface,
			bundleId, userid string) (*model.Consent, error)
		GetBundlesByOwner(ctx contractapi.TransactionContextInterface,
			owner string) ([]*model.Bundle, error)
		GetBreakGlassRequestByRequestID(
			ctx contractapi.TransactionContextInterface,
			requestID string,
			currentUserHash string,
			currentRole string,
		) (*model.BreakGlassRequest, error)
		GetEmergencyBundlesFromDoctor(ctx contractapi.TransactionContextInterface,
			requestID, doctorHash, currentRole string) ([]*model.Bundle, error)
		ApproveBreakGlass(ctx contractapi.TransactionContextInterface,
			requestID, hodHash, decision, hospitalID string) error
		RequestBreakGlass(ctx contractapi.TransactionContextInterface,
			ownerHash, doctorHash, justification, expiresAt string, scopes string, hospitalID, rseq string) (*model.BreakGlassRequest, error)
		ListBreakGlassByDoctor(
			ctx contractapi.TransactionContextInterface,
			doctorHash string,
			pageSize int32,
			bookmark string,
		) (*model.ListBreakGlassRequestResponseWithNextToken, error)
		GetBundlesGrantedToDoctorPaged(
			ctx contractapi.TransactionContextInterface,
			ownerHash string,
			doctorHash string,
			pageSize string,
			bookmark string,
		) (*model.ListBundleResponseWithNextToken, error)
		GetPendingBreakGlassRequests(
			ctx contractapi.TransactionContextInterface, hospitalID string,
		) ([]*model.BreakGlassRequest, error)
		RegisterHospital(
			ctx contractapi.TransactionContextInterface,
			hospitalID string,
			name string,
			createdAt string,
			listIDHoD []string,
		) (*model.Hospital, error)
	}
)

type ConsentContract struct {
	contractapi.Contract
}

func NewConsentService() *ConsentService {
	return &ConsentService{}
}

func now(ctx contractapi.TransactionContextInterface) (*time.Time, error) {
	nowT, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, err
	}

	log.Default().Println("Now*()", nowT.AsTime())
	aTime := nowT.AsTime()
	return &aTime, nil
}

// CreateBundle(id, owner, typ, uri, sha256, size, archetypeId, signature, currentRole, currentUserHash)
func (c *ConsentService) CreateBundle(
	ctx contractapi.TransactionContextInterface,
	id, owner, typ,
	uri, sha256, size, archetypeId,
	signature, currentRole, currentUserHash string,
	hospital *model.Hospital) (*model.Bundle, error) {

	log.Default().Println("CreateBundle", id, owner, typ, uri, sha256, size, archetypeId)

	k := fmt.Sprintf(keyBundle, id)
	exist, _ := ctx.GetStub().GetState(k)
	if len(exist) != 0 {
		return nil, fmt.Errorf("bundle %s exists", id)
	}

	nowT, err := now(ctx)
	if err != nil {
		return nil, err
	}

	b := model.Bundle{
		ID:          id,
		Owner:       owner,
		Type:        typ,
		URI:         uri,
		SHA256:      sha256,
		Size:        size,
		ArchetypeID: archetypeId,
		CreatedAt:   nowT.UTC().Format(time.RFC3339),
		UpdatedAt:   nowT.UTC().Format(time.RFC3339),
		HospitalId:  hospital.HospitalID,
		Signature:   signature,
	}
	val, _ := json.Marshal(b)
	if err := ctx.GetStub().PutState(k, val); err != nil {
		return nil, err
	}

	// index for patient -> bundle
	indexKey, err := ctx.GetStub().CreateCompositeKey(indexPatientBundle, []string{owner, id})
	if err != nil {
		return nil, err
	}
	if err = ctx.GetStub().PutState(indexKey, []byte{0}); err != nil {
		return nil, err
	}

	if model.Role(currentRole) == model.RolesConst.Doctor {
		kConsent := fmt.Sprintf(keyConsent, b.ID, currentUserHash)

		nowT, err = now(ctx)
		if err != nil {
			return nil, err
		}

		cs := model.Consent{}
		cs.SetBundleID(b.ID)
		cs.SetStatus("GRANTED")
		cs.SetGranteeHash(currentUserHash)
		cs.SetPurpose("grant by system")
		cs.SetUpdatedAt(nowT.UTC().Format(time.RFC3339))
		cs.SetCreatedAt(nowT.UTC().Format(time.RFC3339))

		bytesValue, _ := json.Marshal(cs)
		if err = ctx.GetStub().PutState(kConsent, bytesValue); err != nil {
			return nil, err
		}

		compositeKey, err := ctx.GetStub().CreateCompositeKey(indexOwnerConsent, []string{b.Owner, b.ID, currentUserHash})
		if err != nil {
			return nil, fmt.Errorf("[CreateBundle] failed to create composite key: %w", err)
		}

		if err := ctx.GetStub().PutState(compositeKey, []byte{0}); err != nil {
			return nil, fmt.Errorf("failed to store owner index: %w", err)
		}
		result := map[string]string{
			"bundle_id":    b.ID,
			"patient_hash": b.Owner,
			"grantee_hash": currentUserHash,
		}

		if cs["expires_at"] != "" {
			result["expires_at"] = cs["expires_at"]

		}
		payload, _ := json.Marshal(result)
		_ = ctx.GetStub().SetEvent("GrantConsent", payload)
	}

	_ = ctx.GetStub().SetEvent("CreateBundle", val)
	return &b, nil
}

// GrantConsent(bundleId, doctorId, scope, purpose, expiresAt)
func (c *ConsentService) GrantConsent(
	ctx contractapi.TransactionContextInterface,
	bundleId, currentUserHash, currentRole,
	doctorId, purpose, expiresAt string, h *model.Hospital) error {

	log.Default().Println("GrantConsent called with", bundleId, doctorId, purpose, expiresAt, "by user with role", currentRole)
	if model.Role(currentRole) != model.RolesConst.Patient {
		return fmt.Errorf("only patient can grant consent")
	}

	log.Default().Println("GrantConsent", bundleId, doctorId, purpose, expiresAt)

	k := fmt.Sprintf(keyConsent, bundleId, doctorId)

	nowT, err := now(ctx)
	if err != nil {
		return err
	}

	cs := model.Consent{}
	cs.SetBundleID(bundleId)
	cs.SetStatus("GRANTED")
	cs.SetGranteeHash(doctorId)
	cs.SetPurpose(purpose)
	cs.SetUpdatedAt(nowT.UTC().Format(time.RFC3339))
	cs.SetCreatedAt(nowT.UTC().Format(time.RFC3339))

	if expiresAt != "" {
		timeDuration, err := time.ParseDuration(expiresAt)
		if err != nil {
			return err
		}
		expireTime := time.Now().UTC().Add(timeDuration).Format(time.RFC3339)
		cs.SetExpiresTime(expireTime)
	}

	b, err := c.ReadBundle(ctx, bundleId, currentUserHash, currentRole, h)
	if err != nil {
		return fmt.Errorf("cannot create owner index for consent: %w", err)
	}

	log.Default().Println("GrantConsent - bundle loaded:", b)

	if b.Owner != currentUserHash {
		return fmt.Errorf("you can't grant this bundle")
	}

	log.Default().Println("GrantConsent - consent object to store:", cs)

	val, _ := json.Marshal(cs)
	if err := ctx.GetStub().PutState(k, val); err != nil {
		return err
	}

	compositeKey, err := ctx.GetStub().CreateCompositeKey(indexOwnerConsent, []string{b.Owner, bundleId, doctorId})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %w", err)
	}

	fmt.Println("GrantConsent composite key:", compositeKey)
	// value can be empty, we always read actual consent from keyConsent
	if err := ctx.GetStub().PutState(compositeKey, []byte{0}); err != nil {
		return fmt.Errorf("failed to store owner index: %w", err)
	}

	result := map[string]string{
		"bundle_id":    bundleId,
		"patient_hash": b.Owner,
		"grantee_hash": doctorId,
	}
	if cs["expires_at"] != "" {
		result["expires_at"] = cs["expires_at"]

	}
	payload, _ := json.Marshal(result)
	_ = ctx.GetStub().SetEvent("GrantConsent", payload)
	return nil
}

// RevokeConsent(bundleId, doctorId)
func (c *ConsentService) RevokeConsent(ctx contractapi.TransactionContextInterface,
	currentUserId, currentRole, bundleId, doctorId string, h *model.Hospital) error {

	_, err := c.ReadBundle(ctx, bundleId, currentUserId, currentRole, h)
	if err != nil {
		return fmt.Errorf("only bundle owner can revoke consent")
	}

	k := fmt.Sprintf(keyConsent, bundleId, doctorId)
	val, err := ctx.GetStub().GetState(k)
	if err != nil || len(val) == 0 {
		return fmt.Errorf("consent not found")
	}

	nowT, err := now(ctx)
	if err != nil {
		return err
	}
	var cs model.Consent
	_ = json.Unmarshal(val, &cs)
	cs.SetStatus("REVOKED")
	cs.SetUpdatedAt(nowT.UTC().Format(time.RFC3339))

	newVal, _ := json.Marshal(cs)
	if err := ctx.GetStub().PutState(k, newVal); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{
		"bundle_id":    bundleId,
		"grantee_hash": doctorId,
	})

	_ = ctx.GetStub().SetEvent("RevokeConsent", payload)
	return nil
}

func (c *ConsentService) ReadBundle(ctx contractapi.TransactionContextInterface, id, userid, role string, h *model.Hospital) (*model.Bundle, error) {
	k := fmt.Sprintf(keyBundle, id)
	val, err := ctx.GetStub().GetState(k)
	if err != nil || len(val) == 0 {
		return nil, fmt.Errorf("not found")
	}

	var b model.Bundle
	_ = json.Unmarshal(val, &b)

	if b.Owner != userid && model.Role(role) == model.RolesConst.Patient {
		return nil, fmt.Errorf("access denied")
	}

	if model.Role(role) == model.RolesConst.Doctor {
		_, err = c.checkConsent(ctx, b.ID, userid)
		if err != nil {
			bgIter, err := ctx.GetStub().GetStateByPartialCompositeKey(
				indexBGBundle,
				[]string{userid, b.Owner}, // bundleID wildcard
			)
			if err != nil {
				return nil, err
			}

			defer bgIter.Close()
			for bgIter.HasNext() {
				bgkv, err := bgIter.Next()
				if err != nil {
					return nil, err
				}

				_, parts, err := ctx.GetStub().SplitCompositeKey(bgkv.Key)
				if err != nil || len(parts) != 4 {
					return nil, fmt.Errorf("access denied")
				}

				if parts[2] == b.ID {
					requestID := parts[3]
					raw, err := ctx.GetStub().GetState(requestID)
					if err != nil || len(raw) == 0 {
						continue
					}

					var bg model.BreakGlassRequest
					if err = json.Unmarshal(raw, &bg); err != nil {
						continue
					}
					// check expiration
					isExpired, err := onCheckBGExpired(ctx, requestID, &bg)
					if err != nil {
						log.Default().Println("onCheckBGExpired Error:", err)
						continue
					}

					if isExpired {
						log.Default().Println("this bundle has break-glass but it was already expired")
						continue
					}

					return &b, nil
				}
			}
			return nil, err
		}
	}

	return &b, nil
}
func (c *ConsentService) ReadConsent(ctx contractapi.TransactionContextInterface, bundleId, currentUserId, currentRole string, h *model.Hospital) ([]model.Consent, error) {
	log.Default().Println("ReadConsent", bundleId, currentUserId, currentRole)
	if model.Role(currentRole) == model.RolesConst.Doctor {
		k := fmt.Sprintf(keyConsent, bundleId, currentUserId)
		val, err := ctx.GetStub().GetState(k)
		if err != nil || len(val) == 0 {
			return nil, fmt.Errorf("not found")
		}

		var cs model.Consent
		_ = json.Unmarshal(val, &cs)
		return []model.Consent{cs}, nil
	} else if model.Role(currentRole) == model.RolesConst.Patient {
		// Patient can read any consent for their bundles
		b, err := c.ReadBundle(ctx, bundleId, currentUserId, currentRole, h)
		if err != nil {
			return nil, fmt.Errorf("cannot read bundle: %w", err)
		}

		// Load all consents for this bundle
		stub := ctx.GetStub()
		resultsIter, err := stub.GetStateByPartialCompositeKey(indexOwnerConsent, []string{b.Owner, bundleId})
		if err != nil {
			return nil, err
		}
		defer resultsIter.Close()

		var consents []model.Consent
		log.Default().Println(resultsIter, b.Owner, b.ID, bundleId, resultsIter.HasNext())

		for resultsIter.HasNext() {
			kv, err := resultsIter.Next()
			if err != nil {
				return nil, err
			}

			_, parts, err := stub.SplitCompositeKey(kv.Key)
			if err != nil {
				return nil, err
			}
			fmt.Println("read consent composite key parts:", parts)
			if len(parts) != 3 {
				continue
			}
			userId := parts[2]

			consKey := fmt.Sprintf(keyConsent, bundleId, userId)
			consBytes, err := stub.GetState(consKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load consent %s: %w", consKey, err)
			}
			if len(consBytes) == 0 {
				// index without consent (should not happen) → skip
				continue
			}

			var cons model.Consent
			if err := json.Unmarshal(consBytes, &cons); err != nil {
				return nil, fmt.Errorf("invalid consent data for %s: %w", consKey, err)
			}

			consents = append(consents, cons)
		}

		// For simplicity, return the first consent found (if any)
		if len(consents) > 0 {
			return consents, nil
		} else {
			return nil, fmt.Errorf("no consents found for bundle %s", bundleId)
		}
	}

	return nil, fmt.Errorf("access denied")
}

func (c *ConsentService) GetBundlesByOwner(ctx contractapi.TransactionContextInterface,
	owner string) ([]*model.Bundle, error) {
	stub := ctx.GetStub()

	// Query by composite key prefix: (indexName="patient~bundle", keys=[owner])
	resultsIter, err := stub.GetStateByPartialCompositeKey("patient~bundle", []string{owner})
	if err != nil {
		return nil, err
	}
	defer resultsIter.Close()

	var bundles []*model.Bundle

	for resultsIter.HasNext() {
		kv, err := resultsIter.Next()
		if err != nil {
			return nil, err
		}

		// Composite key format: patient~bundle\x00owner\x00bundleID
		_, parts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			continue
		}
		bundleID := parts[1]

		k := fmt.Sprintf(keyBundle, bundleID) // BNDL_<bundleID>
		bundleJSON, err := stub.GetState(k)
		if err != nil {
			return nil, err
		}
		if bundleJSON == nil {
			// index key without actual bundle, skip
			continue
		}

		var b model.Bundle
		if err = json.Unmarshal(bundleJSON, &b); err != nil {
			return nil, err
		}

		bundles = append(bundles, &b)
	}

	return bundles, nil
}

// ListConsentsByOwner(ownerId)
// Returns all *active* consents for bundles owned by ownerId.
func (c *ConsentService) ListConsentsByOwner(ctx contractapi.TransactionContextInterface,
	ownerId, currentRole string, h *model.Hospital) ([]model.ConsentWithBundle, error) {

	stub := ctx.GetStub()
	iter, err := stub.GetStateByPartialCompositeKey(indexOwnerConsent, []string{ownerId})
	if err != nil {
		return nil, fmt.Errorf("failed to query owner index: %w", err)
	}
	defer iter.Close()

	var result []model.ConsentWithBundle

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}

		// composite: ownerId, bundleId, userId
		_, parts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return nil, err
		}
		if len(parts) != 3 {
			continue
		}
		bundleId := parts[1]
		userId := parts[2]

		// Load consent
		consKey := fmt.Sprintf(keyConsent, bundleId, userId)
		consBytes, err := stub.GetState(consKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load consent %s: %w", consKey, err)
		}
		if len(consBytes) == 0 {
			// index without consent (should not happen) → skip
			continue
		}

		var cons model.Consent
		if err := json.Unmarshal(consBytes, &cons); err != nil {
			return nil, fmt.Errorf("invalid consent data for %s: %w", consKey, err)
		}

		if cons.GetExpiresAt() != "" {
			exp, err := time.Parse(time.RFC3339, cons.GetExpiresAt())
			if err == nil && time.Now().UTC().After(exp) {
				continue // expired
			}
		}

		// Load bundle one time per record (for now; you can cache later)
		bdl, err := c.ReadBundle(ctx, bundleId, ownerId, currentRole, h)
		if err != nil {
			return nil, fmt.Errorf("failed to load bundle %s: %w", bundleId, err)
		}

		result = append(result, model.ConsentWithBundle{
			Consent: cons,
			Bundle:  bdl,
		})
	}

	return result, nil
}

func (c *ConsentService) checkConsent(ctx contractapi.TransactionContextInterface,
	bundleId, userid string) (*model.Consent, error) {

	stub := ctx.GetStub()
	k := fmt.Sprintf(keyConsent, bundleId, userid)

	val, err := stub.GetState(k)
	if err != nil {
		return nil, fmt.Errorf("failed to read consent: %w", err)
	}
	if len(val) == 0 {
		return nil, fmt.Errorf("consent not found")
	}

	var cs model.Consent
	if err := json.Unmarshal(val, &cs); err != nil {
		return nil, fmt.Errorf("invalid consent data: %w", err)
	}

	// 1) Status must be GRANTED
	if cs.GetStatus() != "GRANTED" {
		return nil, fmt.Errorf("consent not active: status=%s", cs.GetStatus())
	}

	// 2) If ExpiresAt is set, ensure it's not in the past
	if cs.GetExpiresAt() != "" {
		exp, err := time.Parse(time.RFC3339, cs.GetExpiresAt())
		if err != nil {
			// if parse fails, treat as invalid/expired
			return nil, fmt.Errorf("invalid expiry time on consent")
		}
		if time.Now().UTC().After(exp) {
			cs.SetStatus("EXPIRED")
			bytesData, _ := json.Marshal(cs)
			err = ctx.GetStub().PutState(k, bytesData)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("consent expired at %s", cs.GetExpiresAt())
		}
	}

	// All good → consent is valid
	return &cs, nil
}

func (c *ConsentContract) loadBundleOwner(ctx contractapi.TransactionContextInterface, bundleId string) (string, error) {
	k := fmt.Sprintf(keyBundle, bundleId)
	val, err := ctx.GetStub().GetState(k)
	if err != nil || len(val) == 0 {
		return "", fmt.Errorf("bundle not found")
	}
	var b model.Bundle
	_ = json.Unmarshal(val, &b)
	return b.Owner, nil
}

func (c *ConsentService) GetBundlesGrantedToDoctorPaged(
	ctx contractapi.TransactionContextInterface,
	ownerHash string,
	doctorHash string,
	pageSize string,
	bookmark string,
) (*model.ListBundleResponseWithNextToken, error) {
	var checkDup = make(map[string]struct{})
	result := make([]*model.Bundle, 0)
	nextBookmark, err := DecodeBase64(bookmark)
	if err != nil {
		return nil, err
	}

	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return nil, err
	}
	stub := ctx.GetStub()
	iter, meta, err := stub.GetStateByPartialCompositeKeyWithPagination(
		indexPatientBundle,
		[]string{ownerHash},
		int32(pageSizeInt),
		nextBookmark,
	)
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	// Step 1: get bundles of this owner (paged)

	for iter.HasNext() && len(result) < pageSizeInt {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}

		// patient~bundle => [ownerHash, bundleID]
		_, parts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil || len(parts) != 2 {
			continue
		}
		bundleID := parts[1]

		if _, ok := checkDup[bundleID]; ok {
			continue
		}
		//// Step 2: authoritative consent check
		if _, err := c.checkConsent(ctx, bundleID, doctorHash); err != nil {
			continue // not granted / expired / revoked
		}

		// Step 3: load bundle metadata
		raw, err := stub.GetState(fmt.Sprintf(keyBundle, bundleID))
		if err != nil || len(raw) == 0 {
			continue
		}

		var b model.Bundle
		if err := json.Unmarshal(raw, &b); err != nil {
			continue
		}

		result = append(result, &b)
		checkDup[bundleID] = struct{}{}
	}

	if meta != nil && meta.Bookmark != "" {
		nextBookmark = meta.Bookmark
	}

	if pageSizeInt > len(result) {
		bgIter, err := stub.GetStateByPartialCompositeKey(
			indexBGBundle,
			[]string{doctorHash, ownerHash}, // bundleID wildcard
		)
		if err != nil {
			return nil, err
		}
		defer bgIter.Close()
		for bgIter.HasNext() && len(result) < pageSizeInt {
			bgkv, err := bgIter.Next()
			if err != nil {
				return nil, err
			}

			_, parts, err := stub.SplitCompositeKey(bgkv.Key)
			if err != nil || len(parts) != 4 {
				continue
			}

			bundleID := parts[2]
			requestID := parts[3]

			if _, ok := checkDup[bundleID]; ok {
				log.Default().Println("CheckDup OK", checkDup)
				continue
			}
			bg, err := c.GetBreakGlassRequestByRequestID(
				ctx,
				requestID,
				doctorHash,
				string(model.RolesConst.Doctor),
			)
			if err != nil {
				log.Default().Println("GetBreakGlassRequestByRequestID:595", err.Error())
				continue
			}

			isExpired, err := onCheckBGExpired(ctx, requestID, bg)
			if err != nil {
				log.Default().Println("onCheckBGExpired:595", err.Error())
				continue
			}
			if bg.Status != model.BGApproved || isExpired {
				continue
			}

			raw, err := stub.GetState(fmt.Sprintf(keyBundle, bundleID))
			if err != nil || len(raw) == 0 {
				continue
			}

			var b model.Bundle
			if err := json.Unmarshal(raw, &b); err != nil {
				continue
			}
			checkDup[bundleID] = struct{}{}
			result = append(result, &b)
		}
	}

	return &model.ListBundleResponseWithNextToken{Bundles: result, NextToken: EncodeBase64(nextBookmark)}, nil
}

// ---------- Break-Glass (Emergency) ----------

// RequestBreakGlass(ownerHash, doctorHash, doctorRole, justification, expiresAt, allowedTypesCSV)
// doctorRole must be DOCTOR
func (c *ConsentService) RequestBreakGlass(ctx contractapi.TransactionContextInterface, ownerHash, doctorHash, justification, expiresAt string, scopes string, hospitalID, rseq string) (*model.BreakGlassRequest, error) {

	duration, err := time.ParseDuration(expiresAt)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("expiresAt must be a positive duration")
	}

	getNow, err := now(ctx)
	if err != nil {
		return nil, err
	}
	timeExpiredAt := getNow.Add(duration).UTC().Format(time.RFC3339)
	scopeList := strings.Split(scopes, ",")

	// default emergency allowlist (safe)
	var allowed []model.BundleType
	for _, s := range scopeList {
		bt := model.BundleType(s)
		if model.IsValidBundleType(bt) {
			allowed = append(allowed, bt)
		}
	}

	if len(allowed) == 0 {
		return nil, fmt.Errorf("at least one valid allowedType is required")
	}
	//
	//hasActiveReq, err := hasActiveBreakGlass(ctx, ownerHash)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if hasActiveReq {
	//	return nil, fmt.Errorf("patient already has an active break-glass request")
	//}

	listBundle, err := c.GetBundlesByOwner(ctx, ownerHash)
	if err != nil {
		return nil, err
	}

	if len(listBundle) == 0 {
		return nil, fmt.Errorf("patient has no bundles")
	}

	var (
		mapCheckDupHospital = make(map[string]bool)
		listHospitalIDs     = make([]string, 0)
	)

	isExternal := false
	mapCheckDupHospital[hospitalID] = true
	listHospitalIDs = append(listHospitalIDs, hospitalID)

	for _, b := range listBundle {
		if _, ok := mapCheckDupHospital[b.HospitalId]; !ok {
			mapCheckDupHospital[b.HospitalId] = true
			listHospitalIDs = append(listHospitalIDs, (*b).HospitalId)
		}

		if b.HospitalId != hospitalID {
			isExternal = true
			break
		}
	}

	txTime, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx timestamp: %w", err)
	}

	nowT := time.Unix(txTime.Seconds, int64(txTime.Nanos)).UTC()
	//nowT := time.Now().UTC()
	// enforce allowlist intersection
	bg := &model.BreakGlassRequest{
		OwnerHash:             ownerHash,
		DoctorHash:            doctorHash,
		HODHash:               "-",
		Justification:         justification,
		AllowedTypes:          allowed,
		ExpiresAt:             timeExpiredAt,
		Status:                model.BGRequested,
		CreatedAt:             nowT.Format(time.RFC3339),
		UpdatedAt:             nowT.Format(time.RFC3339),
		BundleIDs:             []string{},
		RequestFromHospitalID: hospitalID,
		IsExternal:            isExternal,
		TotalBundles:          int32(len(listBundle)),
	}

	k := fmt.Sprintf(keyBreakGlass, ownerHash, doctorHash, rseq)
	bg.RequestID = k
	raw, _ := json.Marshal(bg)
	if err := ctx.GetStub().PutState(k, raw); err != nil {
		return nil, err
	}

	err = ctx.GetStub().SetEvent("RequestBreakGlass", raw)
	if err != nil {
		log.Println("RequestBreakGlass Error (line:501):", err)
	}

	// For patients
	if err = putBGOwnerIndex(ctx, ownerHash, k); err != nil {
		log.Println("putBGOwnerIndex Error:", err)
	}

	// For doctors
	if err = putBGDoctorIndex(ctx, doctorHash, k); err != nil {
		log.Println("putBGDoctorIndex Error:", err)
	}

	for _, hid := range listHospitalIDs {
		// for listing pending requests by hospital
		err = putBGStatusIndex(ctx, model.BGRequested, k, hid)
		if err != nil {
			log.Println("putBGStatusIndex Error:", err)
		}

		err = putBGHospitalIndex(ctx, hid, k)
		if err != nil {
			log.Println("putBGHospitalIndex Error:", err)
		}
	}

	return bg, nil
}

func (c *ConsentService) ListBreakGlassByDoctor(
	ctx contractapi.TransactionContextInterface,
	doctorHash string,
	pageSize int32,
	bookmark string,
) (*model.ListBreakGlassRequestResponseWithNextToken, error) {

	bookmarkDecode, err := DecodeBase64(bookmark)
	if err != nil {
		return nil, err
	}

	iter, meta, err := ctx.GetStub().
		GetStateByPartialCompositeKeyWithPagination(
			indexBGDoctor,
			[]string{doctorHash},
			pageSize,
			bookmarkDecode,
		)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var results = make([]*model.BreakGlassRequest, 0)

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}

		_, parts, err := ctx.GetStub().SplitCompositeKey(kv.Key)
		if err != nil || len(parts) < 2 {
			continue
		}

		requestID := parts[1]

		raw, err := ctx.GetStub().GetState(requestID)
		if err != nil || raw == nil {
			continue
		}

		var bg model.BreakGlassRequest
		if err := json.Unmarshal(raw, &bg); err != nil {
			continue
		}

		_, err = onCheckBGExpired(ctx, requestID, &bg)
		if err != nil {
			log.Default().Println("onCheckBGExpired", err)
		}
		results = append(results, &bg)
	}

	nextPageToken := EncodeBase64(meta.Bookmark)

	return &model.ListBreakGlassRequestResponseWithNextToken{Requests: results, NextToken: nextPageToken}, nil
}

// ApproveBreakGlass(ownerHash, doctorHash, hodHash, hodRole, decision)
// hodRole must be HOD. decision ∈ {APPROVE, REJECT}
func (c *ConsentService) ApproveBreakGlass(ctx contractapi.TransactionContextInterface,
	requestID, hodHash, decision, hospitalID string) error {
	check, err := verifyHodHash(ctx, hodHash, hospitalID)
	if err != nil {
		return err
	}

	if !check {
		return fmt.Errorf("hod hash unknown")
	}

	k := requestID
	raw, err := ctx.GetStub().GetState(k)
	if err != nil || len(raw) == 0 {
		log.Println(err)
		return fmt.Errorf("break-glass request not found")
	}

	var bg model.BreakGlassRequest
	err = json.Unmarshal(raw, &bg)
	if err != nil {
		return err
	}

	if bg.Status == model.BGApproved || bg.Status == model.BGRejected {
		return fmt.Errorf("break-glass request already decided: %s", bg.Status)
	} else if bg.Status != model.BGRequested {
		return fmt.Errorf("invalid break-glass request status: %s", bg.Status)
	}

	// cannot approve expired request
	exp, err := time.Parse(time.RFC3339, bg.ExpiresAt)
	if err != nil {
		return err
	}

	nowT, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	if time.Unix(nowT.Seconds, int64(nowT.Nanos)).UTC().After(exp) {
		return fmt.Errorf("break-glass request expired")
	}

	bg.HODHash = hodHash
	switch model.BreakGlassStatus(decision) {
	case model.BGApproved:
		bg.Status = model.BGApproved
	case model.BGRejected:
		bg.Status = model.BGRejected
	default:
		return fmt.Errorf("decision must be APPROVE or REJECT")
	}
	bg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if bg.Status == model.BGApproved {
		allowed := make(map[model.BundleType]bool, len(bg.AllowedTypes))
		for _, t := range bg.AllowedTypes {
			allowed[model.BundleType(strings.ToUpper(string(t)))] = true
		}

		// Load bundles by owner
		bundles, err := c.GetBundlesByOwner(ctx, bg.OwnerHash)
		if err != nil {
			return err
		}

		// Filter to allowed types
		out := make([]string, 0)
		for _, b := range bundles {
			if allowed[model.BundleType(strings.ToUpper(b.Type))] {
				// add to approved list
				err = putBGBundleIndex(ctx, bg.DoctorHash, b.Owner, b.ID, k)
				if err != nil {
					log.Println("putBGBundleIndex Error:", err.Error())
				}
				out = append(out, b.ID)
			}
		}

		bg.BundleIDs = out

		err = deleteBGStatusIndex(ctx, model.BGRequested, hospitalID, k)
		if err != nil {
			log.Println("deleteBGStatusIndex Error:", err)
		}

		err = putBGStatusIndex(ctx, bg.Status, k, hospitalID)
		if err != nil {
			log.Println("putBGStatusIndex Error:", err)
		}
	}

	newRaw, _ := json.Marshal(bg)
	if err := ctx.GetStub().PutState(k, newRaw); err != nil {
		return err
	}

	// Emit event
	log.Println("ApproveBreakGlass event emitted")
	_ = ctx.GetStub().SetEvent("BreakGlassDecision", newRaw)
	return nil
}

// GetEmergencyBundlesByOwner(ownerHash, doctorHash, doctorRole)
// doctorRole must be DOCTOR
func (c *ConsentService) GetEmergencyBundlesFromDoctor(ctx contractapi.TransactionContextInterface,
	requestID, doctorHash, currentRole string) ([]*model.Bundle, error) {

	// Validate break-glass approval
	bgKey := requestID
	raw, err := ctx.GetStub().GetState(bgKey)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("break-glass not found")
	}
	var bg model.BreakGlassRequest
	_ = json.Unmarshal(raw, &bg)

	if model.Role(currentRole) != model.RolesConst.HoD && bg.Status != model.BGApproved {
		return nil, fmt.Errorf("break-glass status=%s", bg.Status)
	}

	isExpired := false
	// check expiration
	isExpired, err = onCheckBGExpired(ctx, requestID, &bg)
	if err != nil {
		log.Default().Println("onCheckBGExpired Error:", err)
	}

	out := make([]*model.Bundle, 0)
	if !isExpired {
		allowed := make(map[model.BundleType]bool, len(bg.AllowedTypes))
		for _, t := range bg.AllowedTypes {
			allowed[model.BundleType(strings.ToUpper(string(t)))] = true
		}

		// Load bundles by owner
		bundles, err := c.GetBundlesByOwner(ctx, bg.OwnerHash)
		if err != nil {
			return nil, err
		}

		// Filter to allowed types
		for _, b := range bundles {
			if allowed[model.BundleType(strings.ToUpper(b.Type))] {

				out = append(out, b)
			}
		}
	}

	// Emit audit event (metadata)
	ev := map[string]any{
		"owner_hash":  bg.OwnerHash,
		"doctor_hash": doctorHash,
		"count":       len(out),
		"ts":          time.Now().UTC().Format(time.RFC3339),
	}
	evRaw, _ := json.Marshal(ev)
	_ = ctx.GetStub().SetEvent("EmergencyReadWithBundles", evRaw)

	return out, nil
}

func (c *ConsentService) GetBreakGlassRequestByRequestID(
	ctx contractapi.TransactionContextInterface,
	requestID string,
	currentUserHash string,
	currentRole string,
) (*model.BreakGlassRequest, error) {

	k := requestID
	raw, err := ctx.GetStub().GetState(k)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("break-glass request not found")
	}

	var bg model.BreakGlassRequest
	_ = json.Unmarshal(raw, &bg)

	// if current user is doctor, must match doctorHash
	// if current user is HOD, can read
	if model.Role(currentRole) == model.RolesConst.Doctor && bg.DoctorHash != currentUserHash {
		return nil, fmt.Errorf("access denied to break-glass request")
	}

	// check expiration
	_, err = onCheckBGExpired(ctx, requestID, &bg)
	if err != nil {
		log.Default().Println("onCheckBGExpired Error:", err)
	}

	ev := map[string]any{
		"owner_hash":  bg.OwnerHash,
		"doctor_hash": bg.DoctorHash,
		"ts":          time.Now().UTC().Format(time.RFC3339),
	}
	evRaw, _ := json.Marshal(ev)
	_ = ctx.GetStub().SetEvent("EmergencyRead", evRaw)
	return &bg, nil
}

func (c *ConsentService) GetPendingBreakGlassRequests(
	ctx contractapi.TransactionContextInterface, hospitalID string,
) ([]*model.BreakGlassRequest, error) {

	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(
		indexBGStatus,
		[]string{string(model.BGRequested), hospitalID},
	)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var result = make([]*model.BreakGlassRequest, 0)

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}

		_, parts, err := ctx.GetStub().SplitCompositeKey(kv.Key)
		if err != nil || len(parts) != 3 {
			continue
		}

		requestID := parts[2]
		raw, err := ctx.GetStub().GetState(requestID)
		if err != nil || len(raw) == 0 {
			continue
		}

		var bg model.BreakGlassRequest
		if err := json.Unmarshal(raw, &bg); err != nil {
			continue
		}
		// check expiration
		_, err = onCheckBGExpired(ctx, requestID, &bg)
		if err != nil {
			log.Default().Println("onCheckBGExpired Error:", err)
			continue
		}

		result = append(result, &bg)
	}

	fmt.Println("GetPendingBreakGlassRequests result count:", len(result), result)

	return result, nil
}

func (c *ConsentService) RegisterHospital(
	ctx contractapi.TransactionContextInterface,
	hospitalID string,
	name string,
	createdAt string,
	listIDHoD []string,
) (*model.Hospital, error) {

	key := fmt.Sprintf(model.KeyHospital, hospitalID)

	// Check existence
	_, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	//if len(existing) > 0 {
	//	return nil, fmt.Errorf("hospital already registered")
	//}

	h := &model.Hospital{
		HospitalID: hospitalID,
		Name:       name,
		Status:     model.HospitalActive,
		CreatedAt:  createdAt,
	}

	bytes, _ := json.Marshal(h)
	if err := ctx.GetStub().PutState(key, bytes); err != nil {
		return nil, err
	}

	for _, id := range listIDHoD {
		err = putHODInHospitalIndex(ctx, id, hospitalID)
		if err != nil {
			log.Default().Println(err.Error())
		}
	}

	// Emit event for FCP sync
	_ = ctx.GetStub().SetEvent("HospitalRegistered", bytes)

	return h, nil
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

// putBGStatusIndex creates BG_STATUS~<status>~<requestID>
func putBGStatusIndex(
	ctx contractapi.TransactionContextInterface,
	status model.BreakGlassStatus,
	requestID string,
	hospitalID string,
) error {
	stub := ctx.GetStub()

	indexKey, err := stub.CreateCompositeKey(
		indexBGStatus,
		[]string{string(status), hospitalID, requestID},
	)
	if err != nil {
		return err
	}

	return stub.PutState(indexKey, []byte{0x00})
}

// deleteBGStatusIndex deletes BG_STATUS~<status>~<requestID>
func deleteBGStatusIndex(
	ctx contractapi.TransactionContextInterface,
	status model.BreakGlassStatus,
	hospitalID string,
	requestID string,
) error {
	stub := ctx.GetStub()

	indexKey, err := stub.CreateCompositeKey(
		indexBGStatus,
		[]string{string(status), hospitalID, requestID},
	)
	if err != nil {
		return err
	}

	return stub.DelState(indexKey)
}

func DecodeBase64(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid next_token")
	}
	return string(raw), nil
}

func EncodeBase64(bookmark string) string {
	return base64.StdEncoding.EncodeToString([]byte(bookmark))
}

func putBGDoctorIndex(
	ctx contractapi.TransactionContextInterface,
	doctorHash string,
	requestID string,
) error {
	key, err := ctx.GetStub().CreateCompositeKey(
		indexBGDoctor,
		[]string{doctorHash, requestID},
	)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte{0x00})
}

func putBGOwnerIndex(
	ctx contractapi.TransactionContextInterface,
	patientHash string,
	requestID string,
) error {
	key, err := ctx.GetStub().CreateCompositeKey(
		indexBGOwner,
		[]string{patientHash, requestID},
	)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte{0x00})
}

func putBGHospitalIndex(
	ctx contractapi.TransactionContextInterface,
	hospitalID string,
	requestID string,
) error {
	key, err := ctx.GetStub().CreateCompositeKey(
		indexBGHospital,
		[]string{hospitalID, requestID},
	)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte{0x00})
}

func putBGBundleIndex(
	ctx contractapi.TransactionContextInterface,
	doctorHash string,
	ownerHash string,
	bundleID string,
	requestID string,
) error {
	key, err := ctx.GetStub().CreateCompositeKey(
		indexBGBundle,
		[]string{doctorHash, ownerHash, bundleID, requestID},
	)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte{0x00})
}

func deleteIndexBGBundle(ctx contractapi.TransactionContextInterface, doctorHash, ownerHash, bundleId string) error {

	stub := ctx.GetStub()

	key, err := stub.GetStateByPartialCompositeKey(indexBGBundle, []string{doctorHash, ownerHash, bundleId})
	if err != nil {
		return err
	}

	defer key.Close()

	for key.HasNext() {
		kv, err := key.Next()
		if err != nil {
			return err
		}
		err = stub.DelState(kv.Key)
		if err != nil {
			return err
		}
	}

	return nil
}

func onCheckBGExpired(
	ctx contractapi.TransactionContextInterface,
	requestID string,
	bg *model.BreakGlassRequest,
) (isExpired bool, err error) {

	exp, err := time.Parse(time.RFC3339, bg.ExpiresAt)
	if err != nil {
		return false, err
	}

	nowT, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return false, err
	}

	if time.Unix(nowT.Seconds, int64(nowT.Nanos)).UTC().After(exp) &&
		(bg.Status == model.BGRequested || bg.Status == model.BGApproved) {
		bg.Status = model.BGExpired
		newRaw, _ := json.Marshal(&bg)
		err = ctx.GetStub().PutState(requestID, newRaw)
		if err != nil {
			log.Default().Println("PutState Error:", err)
			return true, err
		}

		err = deleteBGStatusIndex(ctx, model.BGRequested, bg.RequestFromHospitalID, requestID)
		if err != nil {
			log.Default().Println("deleteBGStatusIndex Error:", err)
			return true, err
		}

		if bg.Status == model.BGApproved {
			err = deleteBGStatusIndex(ctx, model.BGApproved, bg.RequestFromHospitalID, requestID)
			if err != nil {
				log.Default().Println("deleteBGStatusIndex Error:", err)
			}

			// clean up indexBGBundle
			for _, bundleId := range bg.BundleIDs {
				err = deleteIndexBGBundle(ctx, bg.DoctorHash, bg.OwnerHash, bundleId)
				if err != nil {
					log.Default().Println("deleteIndexBGBundle Error:", err)
				}
			}
		}
		_ = ctx.GetStub().SetEvent("BreakGlassExpired", newRaw)
		return true, nil
	}

	return false, nil
}

func hasActiveBreakGlass(
	ctx contractapi.TransactionContextInterface,
	ownerHash string,
) (bool, error) {

	iter, err := ctx.GetStub().
		GetStateByPartialCompositeKey(indexBGOwner, []string{ownerHash})
	if err != nil {
		return false, err
	}
	defer iter.Close()

	getNow, err := now(ctx)
	if err != nil {
		return false, err
	}

	for iter.HasNext() {
		kv, _ := iter.Next()
		_, parts, _ := ctx.GetStub().SplitCompositeKey(kv.Key)
		reqID := parts[1]

		raw, err := ctx.GetStub().GetState(reqID)
		if err != nil || len(raw) == 0 {
			continue
		}

		var bg model.BreakGlassRequest
		_ = json.Unmarshal(raw, &bg)

		if bg.Status == model.BGRequested || bg.Status == model.BGApproved {
			exp, _ := time.Parse(time.RFC3339, bg.ExpiresAt)
			if exp.UTC().After(getNow.UTC()) {
				return true, nil
			}
		}
	}

	return false, nil
}

func putHODInHospitalIndex(
	ctx contractapi.TransactionContextInterface,
	hod string,
	hospitalId string,
) error {
	key, err := ctx.GetStub().CreateCompositeKey(
		indexHoD,
		[]string{hod, hospitalId},
	)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(key, []byte{0x00})
}

func verifyHodHash(ctx contractapi.TransactionContextInterface,
	hod string,
	hospitalId string,
) (bool, error) {
	iter, err := ctx.GetStub().
		GetStateByPartialCompositeKey(indexHoD, []string{})
	if err != nil {
		return false, err
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, _ := iter.Next()
		_, parts, _ := ctx.GetStub().SplitCompositeKey(kv.Key)
		fmt.Println(parts, hod, hospitalId)
		if len(parts) == 2 && hod == parts[0] && hospitalId == parts[1] {
			return true, nil
		}
	}
	return false, nil
}
