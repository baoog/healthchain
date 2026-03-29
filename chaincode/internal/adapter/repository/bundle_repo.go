package repository

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/postgres"
)

type (
	BundleRepoProvider interface {
		// Define methods for bundle repository
		CreateBundle(bundle *entity.Bundle) error
		GetBundleByID(bundleID string) (*entity.Bundle, error)
	}
	BundleRepo struct {
		Postgres *postgres.Client
	}
)

func NewBundleRepo(pgClient *postgres.Client) *BundleRepo {
	return &BundleRepo{
		Postgres: pgClient,
	}
}

func (r *BundleRepo) CreateBundle(bundle *entity.Bundle) error {
	return r.Postgres.DB.Create(bundle).Error
}

func (r *BundleRepo) GetBundleByID(bundleID string) (*entity.Bundle, error) {
	var bundle entity.Bundle
	err := r.Postgres.DB.First(&bundle, "id = ?", bundleID).Error
	if err != nil {
		return nil, err
	}
	return &bundle, nil
}
