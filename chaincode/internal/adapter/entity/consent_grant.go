package entity

import "time"

type ConsentGrant struct {
	ID        uint   `gorm:"primaryKey"`
	BundleID  string `gorm:"index"`
	DoctorID  string `gorm:"index"`
	Status    string `gorm:"size:16"` // GRANTED/REVOKED
	ExpiresAt *time.Time
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type GrantConsentReq struct {
	BundleID  string `json:"bundle_id,omitempty"`
	DoctorID  string `json:"doctor_id,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type RevokeConsentReq struct {
	BundleID string `json:"bundle_id,omitempty"`
	DoctorID string `json:"doctor_id,omitempty"`
}
