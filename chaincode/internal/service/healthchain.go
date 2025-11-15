package service

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/minio"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/infra/client/vault"
	"context"
	"encoding/json"
	"gorm.io/gorm"
)

type HealthChainService struct {
	db     *gorm.DB
	fc     fabric.FabricClient
	minioC minio.IMinoIOClient
	vaultC vault.IVaultClient
}

type HealthChainProvider interface {
	CreateBundle(ctx context.Context, req *entity.CreateBundleReq, currentUserId string) (*entity.Bundle, error)
	//GetBundleByID(ctx context.Context, bundleId string) (*entity.Bundle, error)
	//GetBundlesByOwner(ctx context.Context, ownerId string, limit, offset int) ([]*entity.Bundle, error)
	GrantConsent(ctx context.Context, req *entity.GrantConsentReq, currentUserID, currentRole string) (*entity.ConsentGrant, error)
	//RevokeConsent(ctx context.Context, bundleId, doctorId string) error
	//GetConsent(ctx context.Context, bundleId, doctorId string) (*entity.ConsentGrant, error)
}

func NewHealthChainService(
	dbClient *postgres.Client,
	fabricC fabric.FabricClient,
	minioC minio.IMinoIOClient,
	vaultC vault.IVaultClient,
) *HealthChainService {
	return &HealthChainService{
		db:     dbClient.DB,
		fc:     fabricC,
		minioC: minioC,
		vaultC: vaultC,
	}
}

type EventPayload struct {
	Func string   `json:"func"`
	Args []string `json:"args"`
}

func (s *HealthChainService) CreateBundle(ctx context.Context, req *entity.CreateBundleReq, currentUserId string) (*entity.Bundle, error) {
	req.Owner = currentUserId
	res, err := s.fc.Contract().SubmitTransaction(
		"CreateBundle",
		req.ID, req.Owner, req.Type, req.URI, req.SHA256, req.Size, req.ArchetypeID,
	)
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

func (s *HealthChainService) GrantConsent(ctx context.Context, req *entity.GrantConsentReq, currentUserID, currentRole string) (*entity.ConsentGrant, error) {
	res, err := s.fc.Contract().SubmitTransaction(
		"GrantConsent",
		req.BundleID, currentUserID, currentRole, req.DoctorID, req.Scope, req.Purpose, req.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	var consent entity.ConsentGrant
	err = json.Unmarshal(res, &consent)
	if err != nil {
		return nil, err
	}

	return &consent, nil
}
