package service

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/adapter/repository"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"time"
)

type EventConsumer struct {
	fc            fabric.FabricClient
	bundleRepo    repository.BundleRepoProvider
	consentRepo   repository.ConsentRepoProvider
	emergencyRepo repository.EmergencyAccessRepoProvider
	service       *HealthChainService
	*postgres.Client
}

type IEventConsumer interface {
	Start(ctx context.Context)
}

func NewEventConsumer(
	fc fabric.FabricClient,
	s *HealthChainService,
	db *postgres.Client,
	bundleRepo repository.BundleRepoProvider,
	consentRepo repository.ConsentRepoProvider,
	emergencyRepo repository.EmergencyAccessRepoProvider) *EventConsumer {
	return &EventConsumer{fc: fc, service: s, Client: db, bundleRepo: bundleRepo, consentRepo: consentRepo, emergencyRepo: emergencyRepo}
}

func (c *EventConsumer) Start(ctx context.Context) {
	log.Default().Println("Starting event consumer...")
	evCh, cancel, err := c.fc.ContractEvents(ctx)
	if err != nil {
		log.Fatalf("contract events: %v", err)
	}
	go func() {
		defer cancel()
		for {
			select {
			case ev, ok := <-evCh:
				if !ok || ctx.Err() != nil {
					return
				}

				type FabricEvent struct {
					EventName     string   `json:"event_name"`
					TransactionID string   `json:"transaction_id"`
					Args          []string `json:"args"`
				}
				var evData FabricEvent
				err = json.Unmarshal(ev.Payload, &evData)
				if err != nil {
					log.Printf("Error unmarshaling event payload: %v", err)
					continue
				}

				switch ev.EventName {
				case "CreateBundle":
					bundle := entity.Bundle{}
					err = json.Unmarshal(ev.Payload, &bundle)
					if err != nil {
						log.Println("Error unmarshaling CreateBundle event payload: ", err)
						continue
					}

					err = c.bundleRepo.CreateBundle(&bundle)
					if err != nil {
						log.Printf("Error saving bundle from event: %v", err)
					}

				case "GrantConsent":
					type GrantConsentEvent struct {
						BundleID    string `json:"bundle_id"`
						PatientHash string `json:"patient_hash"`
						GranteeHash string `json:"grantee_hash"`
						ExpiresAt   string `json:"expires_at"`
						Scope       string `json:"scope"`
					}
					var payload GrantConsentEvent
					err = json.Unmarshal(ev.Payload, &payload)
					if err != nil {
						log.Println("Error unmarshaling GrantConsent event payload: ", err)
						return
					}

					cs := &entity.Consent{
						ID:          uuid.NewString(),
						BundleID:    payload.BundleID,
						PatientHash: payload.PatientHash,
						GranteeHash: payload.GranteeHash,
						Status:      model.ConsentActionConst.Granted,
						ChainTxID:   ev.TransactionID,
					}

					if payload.ExpiresAt != "" {
						expiredAt, err := time.ParseInLocation(time.RFC3339, payload.ExpiresAt, time.UTC)
						if err != nil {
							log.Printf("Error parsing ExpiresAt in GrantConsent event: %v", err)
							continue
						}

						cs.ExpiresAt = expiredAt
					}
					c.consentRepo.CreateConsent(cs)

				case "RevokeConsent":
					type RevokeConsentEvent struct {
						BundleID    string `json:"bundle_id"`
						GranteeHash string `json:"grantee_hash"`
					}
					var payload RevokeConsentEvent

					err = json.Unmarshal(ev.Payload, &payload)
					if err != nil {
						log.Printf("Error unmarshaling RevokeConsent event payload: %v", err)
						continue
					}

					err = c.consentRepo.RevokeConsentByBundleAndDoctor(
						payload.BundleID, payload.GranteeHash,
					)
					if err != nil {
						log.Printf("Error revoking consent from event: %v", err)
					}

				case "RequestBreakGlass":
					fmt.Println(string(ev.Payload))
					type BreakGlassEvent struct {
						RequestId    string   `json:"request_id"`
						OwnerHash    string   `json:"owner_hash"`
						DoctorHash   string   `json:"doctor_hash"`
						Reason       string   `json:"reason"`
						ExpiresAt    string   `json:"expires_at"`
						AllowedTypes []string `json:"allowed_types"`
						CreatedAt    string   `json:"created_at"`
						UpdatedAt    string   `json:"updated_at"`
					}
					var payload BreakGlassEvent
					err = json.Unmarshal(ev.Payload, &payload)
					if err != nil {
						log.Printf("Error unmarshaling BreakGlass event payload: %v", err)
						continue
					}

					expireAt, err := time.ParseInLocation(time.RFC3339, payload.ExpiresAt, time.UTC)
					if err != nil {
						log.Printf("Error parsing expireAt in BreakGlass event: %v", err)
						continue
					}

					allowedTypes := entity.ToListBundleType(payload.AllowedTypes)

					err = c.emergencyRepo.CreateEmergencyAccessRequest(context.Background(), &entity.EmergencyAccess{
						ID:            uuid.NewString(),
						RequestID:     payload.RequestId,
						OwnerHash:     payload.OwnerHash,
						DoctorHash:    payload.DoctorHash,
						HODHash:       "-",
						Justification: payload.Reason,
						Status:        string(model.EmergencyAccessStatusConst.Requested),
						ExpiresAt:     expireAt,
						CreatedAt:     payload.CreatedAt,
						UpdatedAt:     payload.UpdatedAt,
						AllowedTypes:  allowedTypes,
					})

				case "BreakGlassDecision":
					type BreakGlassDecisionEvent struct {
						RequestId string    `json:"request_id"`
						HODHash   string    `json:"hod_hash"`
						Status    string    `json:"status"`
						BundleIDs []string  `json:"bundle_ids"`
						UpdatedAt time.Time `json:"updated_at"`
					}

					var payload BreakGlassDecisionEvent
					err = json.Unmarshal(ev.Payload, &payload)
					if err != nil {
						log.Printf("Error unmarshaling BreakGlassDecision event payload: %v", err)
						continue
					}

					err = c.emergencyRepo.UpdateEmergencyAccessDecision(context.Background(),
						payload.RequestId,
						payload.HODHash,
						payload.Status,
						payload.UpdatedAt,
						payload.BundleIDs,
					)
					if err != nil {
						log.Printf("Error updating BreakGlassDecision from event: %v", err)
					}
					fmt.Println("Received BreakGlassDecision event: ", payload)

				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
