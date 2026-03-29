package repository

import (
	"context"
	"fcp/internal/adapter/entity"
	"fcp/internal/infra/client/postgres"
	"github.com/google/uuid"
)

type KeyRepository interface {
	FindActiveByHospitalID(ctx context.Context, hospitalUUID uuid.UUID) (*entity.HospitalKey, error)
	AddKeyHistory(ctx context.Context, hk *entity.HospitalKey) error
	UpdateKey(ctx context.Context, hk *entity.HospitalKey) error
}

type keyRepo struct {
	pq *postgres.Client
}

func NewKeyRepo(db *postgres.Client) KeyRepository {
	return &keyRepo{pq: db}
}

func (r *keyRepo) FindActiveByHospitalID(ctx context.Context, hospitalUUID uuid.UUID) (*entity.HospitalKey, error) {
	var k entity.HospitalKey
	err := r.pq.GetDB(ctx).
		Where("hospital_id = ? AND is_active = true", hospitalUUID).
		Order("version DESC").
		First(&k).Error
	return &k, err
}

func (r *keyRepo) AddKeyHistory(ctx context.Context, hk *entity.HospitalKey) error {
	return r.pq.GetDB(ctx).Create(hk).Error
}

func (r *keyRepo) UpdateKey(ctx context.Context, hk *entity.HospitalKey) error {
	return r.pq.GetDB(ctx).Save(hk).Error
}
