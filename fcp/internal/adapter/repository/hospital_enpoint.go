package repository

import (
	"context"
	"fcp/internal/adapter/entity"
	"fcp/internal/infra/client/postgres"
	"github.com/google/uuid"
)

type endpointRepo struct {
	pq *postgres.Client
}

type EndpointRepository interface {
	FindPrimaryByHospitalID(ctx context.Context, hospitalUUID uuid.UUID) (*entity.HospitalEndpoint, error)
	CreateEndpoint(ctx context.Context, ep *entity.HospitalEndpoint) error
}

func NewEndpointRepo(db *postgres.Client) EndpointRepository {
	return &endpointRepo{pq: db}
}

func (r *endpointRepo) FindPrimaryByHospitalID(ctx context.Context, hospitalUUID uuid.UUID) (*entity.HospitalEndpoint, error) {
	var ep entity.HospitalEndpoint
	err := r.pq.GetDB(ctx).
		Where("hospital_id = ? AND is_primary = true", hospitalUUID).
		First(&ep).Error
	return &ep, err
}

func (r *endpointRepo) CreateEndpoint(ctx context.Context, ep *entity.HospitalEndpoint) error {
	return r.pq.GetDB(ctx).Create(ep).Error
}
