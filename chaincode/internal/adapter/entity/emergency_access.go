package entity

import (
	"time"
)

type EmergencyAccess struct {
	ID            string             `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"` // UUIDv4
	RequestID     string             `json:"request_id" gorm:"size:256;uniqueIndex;not null"`          // unique request id
	OwnerHash     string             `json:"owner_hash" gorm:"size:128;index;not null"`                // hashed patient/user id (owner)
	DoctorHash    string             `json:"doctor_hash" gorm:"size:128;index;not null"`               // hashed doctor/user id
	HODHash       string             `json:"hod_hash" gorm:"size:128;index"`                           // hashed HoD id (filled on approval)
	Justification string             `json:"justification" gorm:"type:text;not null"`                  // short reason
	Status        string             `json:"status" gorm:"size:16;index;not null"`                     // REQUESTED | APPROVED | REJECTED | EXPIRED
	ExpiresAt     time.Time          `json:"expires_at"`                                               // RFC3339
	CreatedAt     string             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     string             `json:"updated_at" gorm:"autoUpdateTime"`
	AllowedTypes  BundleTypeArray    `json:"allowed_types,omitempty" gorm:"type:_text"`
	Bundles       []BreakGlassBundle `json:"bundles" gorm:"foreignKey:RequestID;references:RequestID"`
	IsExternal    *bool              `json:"is_external,omitempty" gorm:"-:all"`
}

type EmergencyAccessDecisionReq struct {
	RequestID string `json:"request_id"`
	HoDHash   string `json:"hod_hash"`
	Decision  string `json:"decision"`
}

type EmergencyAccessRequest struct {
	PatientHash      string   `json:"patient_id"`
	Justification    string   `json:"justification"`
	ExpiresInMinutes string   `json:"expires_in_minutes"`
	Types            []string `json:"types"` // ["ALLERGY",...]
}

type BreakGlassBundle struct {
	ID        string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	RequestID string `gorm:"size:64;index;not null"`
	BundleID  string `gorm:"size:64;index;not null"` // FK → bundles.id
}

func (EmergencyAccess) TableName() string {
	return "break_glass_requests"
}

func (BreakGlassBundle) TableName() string {
	return "break_glass_bundles"
}

type BreakGlassRequestContract struct {
	RequestID             string       `json:"request_id"`
	OwnerHash             string       `json:"owner_hash"`
	RequestFromHospitalID string       `json:"request_from_hospital_id"`
	DoctorHash            string       `json:"doctor_hash"`
	HODHash               string       `json:"hod_hash,omitempty"`
	Justification         string       `json:"justification"`
	AllowedTypes          []BundleType `json:"allowed_types,omitempty"`
	ExpiresAt             string       `json:"expires_at"` // RFC3339
	Status                string       `json:"status"`
	CreatedAt             string       `json:"created_at"`
	UpdatedAt             string       `json:"updated_at"`
	BundleIDs             []string     `json:"bundle_ids"`
	IsExternal            bool         `json:"is_external"`
	TotalBundles          int32        `json:"total_bundles"`
	Owner                 *User        `json:"owner,omitempty"`
}

type ListBreakGlassRequestResponseWithNextToken struct {
	Requests  []*BreakGlassRequestContract `json:"requests"`
	NextToken string                       `json:"next_token"`
}
