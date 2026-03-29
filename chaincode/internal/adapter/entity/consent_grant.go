package entity

import (
	"consentcc/internal/model"
	"time"
)

type (
	Consent struct {
		ID          string              `json:"id,omitempty" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
		BundleID    string              `json:"bundle_id,omitempty" gorm:"column:bundle_id;index;not null"`
		PatientHash string              `json:"patient_hash,omitempty" gorm:"column:patient_hash;type:varchar(64);index;not null"`
		GranteeHash string              `json:"grantee_hash,omitempty" gorm:"column:grantee_hash;type:varchar(64);index;not null"`
		Status      model.ConsentAction `json:"status,omitempty" gorm:"column:status;type:varchar(20)"` // GRANTED/REVOKED
		Purpose     string              `json:"purpose,omitempty" gorm:"column:purpose;type:text"`
		ExpiresAt   time.Time           `json:"expires_at,omitempty"`
		CreatedAt   time.Time           `json:"created_at,omitempty" gorm:"autoCreateTime"`
		UpdatedAt   time.Time           `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
		ChainTxID   string              `json:"chain_tx_id,omitempty" gorm:"column:chain_tx_id;type:varchar(128)"`
	}
	GrantConsentReq struct {
		BundleIDs []string `json:"bundle_ids,omitempty"`
		DoctorID  string   `json:"doctor_id,omitempty"`
		Purpose   string   `json:"purpose,omitempty"`
		ExpiresAt string   `json:"expires_at,omitempty"`
	}

	RevokeConsentReq struct {
		BundleID string `json:"bundle_id,omitempty"`
		DoctorID string `json:"doctor_id,omitempty"`
	}
)
