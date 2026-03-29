package repository

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/model"
	"fmt"
)

type (
	ConsentRepoProvider interface {
		// Define methods for consent repository
		CreateConsent(consent *entity.Consent) error
		GetConsentByBundleAndDoctor(bundleID, doctorID string) (*entity.Consent, error)
		RevokeConsentByBundleAndDoctor(bundleID, doctorID string) error
	}
	ConsentRepo struct {
		// Implement consent repository methods here
		Postgres *postgres.Client
	}
)

func NewConsentRepo(pgClient *postgres.Client) *ConsentRepo {
	return &ConsentRepo{
		Postgres: pgClient,
	}
}

func (r *ConsentRepo) CreateConsent(consent *entity.Consent) error {
	if consent == nil {
		return fmt.Errorf("consent is nil")
	}

	consent.Status = model.ConsentActionConst.Granted
	return r.Postgres.DB.Create(consent).Error
}

func (r *ConsentRepo) GetConsentByBundleAndDoctor(bundleID, doctorID string) (*entity.Consent, error) {
	var consent entity.Consent
	err := r.Postgres.DB.First(&consent, "bundle_id = ? AND grantee_hash = ?", bundleID, doctorID).Error
	if err != nil {
		return nil, err
	}
	return &consent, nil
}

func (r *ConsentRepo) RevokeConsentByBundleAndDoctor(bundleID, doctorID string) error {
	return r.Postgres.DB.Model(&entity.Consent{}).
		Where("bundle_id = ? AND grantee_hash = ?", bundleID, doctorID).
		Update("status", model.ConsentActionConst.Revoked).Error
}
