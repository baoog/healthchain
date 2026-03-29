package repository

import (
	"context"
	"fcp/internal/adapter/entity"
	"fcp/internal/infra/client/postgres"
)

type HospitalRepository interface {
	Create(ctx context.Context, h *entity.Hospital) error
	GetByHospitalID(ctx context.Context, hospitalID string) (*entity.Hospital, error)
	Update(ctx context.Context, h *entity.Hospital) error
	WithTx(ctx context.Context, txFn func(ctx2 context.Context) error) error
}

type hospitalRepo struct {
	pq *postgres.Client
}

func NewHospitalRepo(db *postgres.Client) HospitalRepository {
	return &hospitalRepo{pq: db}
}

func (r *hospitalRepo) Create(ctx context.Context, h *entity.Hospital) error {
	return r.pq.GetDB(ctx).Create(h).Error
}

func (r *hospitalRepo) GetByHospitalID(ctx context.Context, hospitalID string) (*entity.Hospital, error) {
	var h entity.Hospital
	if err := r.pq.GetDB(ctx).
		Where("hospital_id = ?", hospitalID).
		First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *hospitalRepo) Update(ctx context.Context, h *entity.Hospital) error {
	return r.pq.GetDB(ctx).Save(h).Error
}

func (r *hospitalRepo) WithTx(ctx context.Context, txFn func(ctx2 context.Context) error) error {
	return r.pq.WithTx(ctx, txFn)
}
