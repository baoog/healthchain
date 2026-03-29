package repository

import (
	"context"
	"fcp/internal/adapter/entity"
	"fcp/internal/infra/client/postgres"
	"github.com/google/uuid"
)

type federationInterfaceRepo struct {
	pq *postgres.Client
}

type FederationInterfaceRepository interface {
	FindByHospitalAndName(
		ctx context.Context,
		hospitalUUID uuid.UUID,
		name entity.FederationInterfaceName,
	) (*entity.FederationInterface, error)
	CreateInterFace(ctx context.Context, fi []*entity.FederationInterface) error
}

func NewFederationInterfaceRepo(pq *postgres.Client) FederationInterfaceRepository {
	return &federationInterfaceRepo{pq: pq}
}

func (r *federationInterfaceRepo) FindByHospitalAndName(ctx context.Context,
	hospitalUUID uuid.UUID,
	name entity.FederationInterfaceName,
) (*entity.FederationInterface, error) {

	var fi entity.FederationInterface
	db := r.pq.GetDB(ctx)
	err := db.Debug().
		Where("hospital_id = ? AND name = ?", hospitalUUID, name).
		First(&fi).Error
	if err != nil {
		return nil, err
	}
	return &fi, nil
}

func (r *federationInterfaceRepo) CreateInterFace(ctx context.Context, fi []*entity.FederationInterface) error {
	return r.pq.GetDB(ctx).Create(fi).Error
}
