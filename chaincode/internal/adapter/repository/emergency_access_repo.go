package repository

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/postgres"
	"context"
	"github.com/google/uuid"
	"time"
)

type EmergencyAccessRepoProvider interface {
	// Define methods for emergency access repository
	CreateEmergencyAccessRequest(ctx context.Context, req *entity.EmergencyAccess) error
	UpdateEmergencyAccessDecision(ctx context.Context, requestID, hodHash, status string, updatedAt time.Time, bundles []string) error
}

type EmergencyAccessRepo struct {
	// Add necessary fields, e.g., database client
	Postgres *postgres.Client
}

func NewEmergencyAccessRepo(pgClient *postgres.Client) *EmergencyAccessRepo {
	return &EmergencyAccessRepo{
		Postgres: pgClient,
	}
}

func (r *EmergencyAccessRepo) CreateEmergencyAccessRequest(ctx context.Context, req *entity.EmergencyAccess) error {
	// Implement the logic to create an emergency access request in the database
	res := r.Postgres.DB.WithContext(ctx).Create(req)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *EmergencyAccessRepo) UpdateEmergencyAccessDecision(ctx context.Context, requestID, hodHash, status string, updatedAt time.Time, bundles []string) error {
	// Implement the logic to update the emergency access decision in the database
	updateData := map[string]interface{}{
		"status":     status,
		"hod_hash":   hodHash,
		"updated_at": updatedAt,
	}
	res := r.Postgres.DB.WithContext(ctx).Model(&entity.EmergencyAccess{}).Where("request_id = ?", requestID).Updates(updateData)
	if res.Error != nil {
		return res.Error
	}

	bgWithBundles := []entity.BreakGlassBundle{}
	for _, id := range bundles {
		bgWithBundles = append(bgWithBundles, entity.BreakGlassBundle{
			BundleID:  id,
			ID:        uuid.NewString(),
			RequestID: requestID,
		})
	}

	res = r.Postgres.DB.WithContext(ctx).Model(&entity.BreakGlassBundle{}).Create(&bgWithBundles)
	if res.Error != nil {
		return res.Error
	}
	return nil

}
