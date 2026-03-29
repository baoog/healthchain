package service

import (
	"consentcc/config"
	"consentcc/internal/adapter/entity"
	"consentcc/internal/adapter/repository"
	"consentcc/internal/infra/client/fcp"
	"consentcc/internal/infra/client/http"
	"consentcc/internal/infra/client/minio"
	"consentcc/internal/infra/client/vault"
	"consentcc/internal/model"
	"consentcc/utils"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"io"
	"log"
	"strings"
	"time"
)

type (
	OrchestratorServiceProvider interface {
		// Define methods for orchestrator service
		GetBundlesDetail(ctx context.Context, bundles []*entity.BundleFromFabric, currentUserHash, currentRole string) ([]*entity.BundleFromFabric, error)
		GetUsersInfoForBundles(ctx context.Context, bundles []*entity.BundleFromFabric) (*[]entity.BundleWithUser, error)
	}
	OrchestratorService struct {
		// Add necessary fields here
		vaultClient *vault.Client
		minioClient *minio.Client
		userRepo    repository.UserRepoProvider
		fcpClient   fcp.Client
		apiKey      string
	}
)

func NewOrchestratorService(
	vaultClient *vault.Client,
	minioClient *minio.Client,
	userRepo repository.UserRepoProvider,
	fcpClient fcp.Client,
	cfg *config.Config,
) *OrchestratorService {
	return &OrchestratorService{
		vaultClient: vaultClient,
		minioClient: minioClient,
		userRepo:    userRepo,
		fcpClient:   fcpClient,
		apiKey:      cfg.APIKey,
	}
}

func (s *OrchestratorService) GetBundlesDetail(ctx context.Context, bundles []*entity.BundleFromFabric, currentUserHash, currentRole string) ([]*entity.BundleFromFabric, error) {
	// Implement the logic to get bundle details
	for _, bundle := range bundles {
		if bundle.URI != "" && bundle.IsExternal != nil && !*bundle.IsExternal {
			log.Default().Println("Internal")
			rc, err := s.minioClient.GetObject(ctx, bundle.URI)
			if err != nil {
				log.Default().Println("Error retrieving bundle from MinIO:", err)
				return nil, fmt.Errorf("failed to retrieve bundle from storage: %w", err)
			}

			defer rc.Close()

			encBytes, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read bundle content: %w", err)
			}

			actualHash := utils.SHA256Hex(encBytes)
			if bundle.SHA256 != "" && !strings.EqualFold(bundle.SHA256, actualHash) {
				return nil, fmt.Errorf("data integrity check failed for bundle ID %s", bundle.ID)
			}

			// 3️⃣ Authenticity check: verify Ed25519 signature BEFORE decrypt
			// hash must be verified as raw bytes (not hex string)
			hashBytes := sha256.Sum256(encBytes)
			if bundle.Signature == "" || bundle.HospitalId == "" {
				return nil, fmt.Errorf("missing signature metadata for bundle ID %s", bundle.ID)
			}

			fmt.Println("Stored SHA256:", bundle.SHA256)
			fmt.Println("Recomputed SHA256:", utils.SHA256Hex(encBytes))
			fmt.Println("Signature:", bundle.Signature)

			ok, err := s.vaultClient.VerifySignatureOfBundle(ctx, hashBytes[:], bundle.Signature)
			if err != nil {
				return nil, err
			}

			if !ok {
				return nil, fmt.Errorf("signature verification failed for bundle id %s", bundle.ID)
			}

			aesKey, err := s.vaultClient.GetBundleKey(ctx, bundle.ID, "ehr/keys")
			if err != nil {
				return nil, fmt.Errorf("failed to get AES key from vault: %w", err)
			}

			decBytes, err := utils.DecryptAESGCM(aesKey, encBytes)
			if err != nil {
				log.Default().Println(err.Error())
				return nil, fmt.Errorf("failed to decrypt bundle content: %w", err)
			}

			var data any
			if err := json.Unmarshal(decBytes, &data); err != nil {
				return nil, fmt.Errorf("failed to parse bundle content: %w", err)
			}
			bundle.Data = data
		} else if bundle.URI != "" && bundle.IsExternal != nil && *bundle.IsExternal {
			log.Default().Println("External")
			log.Default().Println("Call FCP")
			respResv, err := s.fcpClient.ResolveHospital(ctx, bundle.HospitalId, fcp.InterfaceGetBundleDetail)
			if err != nil {
				return nil, err
			}

			log.Default().Println("Call FCP", "done")

			pubKeyBytes, err := base64.StdEncoding.DecodeString(respResv.Key.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("invalid hospital public key: %w", err)
			}

			if len(pubKeyBytes) != ed25519.PublicKeySize {
				return nil, fmt.Errorf("invalid ed25519 public key length")
			}

			pubKey := ed25519.PublicKey(pubKeyBytes)

			httpRequest := http.NewClient(respResv.Endpoint.BaseURL, s.apiKey)

			var modelRequestGetBundle fcp.GetBundleDetailRequest
			bytes, _ := respResv.Interface.Request.MarshalJSON()
			err = json.Unmarshal(bytes, &modelRequestGetBundle)
			if err != nil {
				return nil, err
			}

			var (
				bodyMap = map[string]interface{}{
					modelRequestGetBundle.FieldBundleID: bundle.ID,
				}
				respMap = make(map[string]interface{})
			)

			log.Default().Println(currentRole)

			if currentRole == string(model.RolesConst.Doctor) {
				bodyMap[modelRequestGetBundle.FieldRequesterDoctorHash] = currentUserHash
			} else if currentRole == string(model.RolesConst.Patient) {
				bodyMap[modelRequestGetBundle.FieldRequesterPatientHash] = currentUserHash
			}
			log.Default().Println(bodyMap)
			log.Default().Println("Call to target endpoint")
			err = httpRequest.DoRequest(ctx, respResv.Interface.Method, respResv.Interface.Path, bodyMap, &respMap)
			if err != nil {
				return nil, err
			}

			log.Default().Println("Call to target endpoint", "done")
			log.Default().Println(respMap)
			// 4️⃣ Extract signed payload
			payloadB64, ok := respMap["payload"].(string)
			if !ok {
				return nil, fmt.Errorf("missing payload_b64")
			}

			signatureB64, ok := respMap["signature"].(string)
			if !ok {
				return nil, fmt.Errorf("missing signature_b64")
			}

			signParts := strings.Split(signatureB64, ":")
			if len(signParts) != 3 {
				return nil, fmt.Errorf("invalid vault signature format")
			}

			rawSigBase64 := signParts[2]

			// 5️⃣ Decode
			payloadBytes, err := base64.StdEncoding.DecodeString(payloadB64)
			if err != nil {
				return nil, fmt.Errorf("decode payload failed: %w", err)
			}

			signatureBytes, err := base64.StdEncoding.DecodeString(rawSigBase64)
			if err != nil {
				return nil, fmt.Errorf("decode signature failed: %w", err)
			}

			// 6️⃣ Verify signature using Hospital B public key
			if !ed25519.Verify(pubKey, payloadBytes, signatureBytes) {
				return nil, fmt.Errorf("signature verification failed (hospital %s)", bundle.HospitalId)
			}
			var signedPayload struct {
				OK     bool                     `json:"ok,omitempty"`
				Bundle *entity.BundleFromFabric `json:"bundle,omitempty"`
				User   *entity.User             `json:"user,omitempty"`
			}

			if err := json.Unmarshal(payloadBytes, &signedPayload); err != nil {
				return nil, fmt.Errorf("invalid signed payload: %w", err)
			}

			if signedPayload.Bundle != nil {
				bundle.Data = signedPayload.Bundle.Data
			}
			bundle.User = signedPayload.User
		}
	}
	return bundles, nil
}

func (s *OrchestratorService) GetUsersInfoForBundles(ctx context.Context, bundles []*entity.BundleFromFabric) (*[]entity.BundleWithUser, error) {
	var (
		listNationalHash []string
		checkDup         = make(map[string]struct{})
	)
	for _, b := range bundles {
		if b != nil {
			if _, ok := checkDup[(*b).Owner]; ok {
				continue
			}
			checkDup[(*b).Owner] = struct{}{}
			listNationalHash = append(listNationalHash, (*b).Owner)
		}

	}

	_, mapUser, err := s.userRepo.LoadUsersByListNationalHash(ctx, listNationalHash)
	if err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, sql.ErrNoRows)) {
		return nil, err
	}

	var listBundleWithUser []entity.BundleWithUser
	for _, b := range bundles {
		createdAtTime, err := time.ParseInLocation(time.RFC3339Nano, b.CreatedAt, time.UTC)
		if err != nil {
			return nil, err
		}

		bundleWithUser := entity.BundleWithUser{
			ID:          b.ID,
			OwnerHash:   b.Owner,
			Type:        entity.ToBundleType(b.Type),
			ObjectURI:   b.URI,
			SHA256:      b.SHA256,
			Size:        b.Size,
			ArchetypeID: b.ArchetypeID,
			Data:        nil,
			CreatedAt:   &createdAtTime,
		}
		True := true
		if userData, ok := mapUser[b.Owner]; ok {
			bundleWithUser.User = userData.User
			if b.HospitalId != "" && entity.GetHospitalID() != b.HospitalId {
				bundleWithUser.IsExternal = &True
			} else if b.HospitalId != "" && entity.GetHospitalID() == b.HospitalId {
				False := false
				bundleWithUser.IsExternal = &False
			}

		} else {
			bundleWithUser.User = &entity.User{Name: "Unknown"}
			bundleWithUser.IsExternal = &True
		}
		listBundleWithUser = append(listBundleWithUser, bundleWithUser)
	}

	return &listBundleWithUser, nil
}
