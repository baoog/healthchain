package entity

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type HospitalStatus string
type FederationInterfaceName string

const (
	HospitalActive   HospitalStatus = "ACTIVE"
	HospitalDisabled HospitalStatus = "DISABLED"
)

type Hospital struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	HospitalID       string         `gorm:"size:64;uniqueIndex;not null"` // "hospital-b"
	Name             string         `gorm:"size:255;not null"`
	Status           HospitalStatus `gorm:"size:16;index;not null"`
	ClientID         string         `gorm:"size:64;uniqueIndex;not null"`
	ClientSecretHash string         `gorm:"type:text;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type HospitalFabric struct {
	HospitalID string         `json:"hospital_id"` // 64-char, deterministic
	Name       string         `json:"name"`
	Status     HospitalStatus `json:"status"`
	CreatedAt  string         `json:"created_at"` // RFC3339
}

type HospitalEndpoint struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	HospitalID uuid.UUID `gorm:"type:uuid;index;not null"`

	APIBaseURL string `gorm:"type:text;not null"` // https://orch.hospital-b.example.com
	APIVersion string `gorm:"size:16;not null"`   // v1

	IsPrimary bool `gorm:"default:false"`

	CreatedAt time.Time
}

type HospitalKey struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	HospitalID uuid.UUID `gorm:"type:uuid;index;not null"`

	KeyType   string `gorm:"size:16;not null"` // RSA
	PublicKey string `gorm:"type:text;not null"`
	Version   int    `gorm:"not null;index"`

	IsActive bool `gorm:"default:true"`

	CreatedAt time.Time
}

func (s HospitalStatus) Value() (driver.Value, error) {
	switch s {
	case HospitalActive, HospitalDisabled:
		return string(s), nil
	default:
		return nil, fmt.Errorf("invalid HospitalStatus: %s", s)
	}
}

func (s *HospitalStatus) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("HospitalStatus cannot be NULL")
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("HospitalStatus must be string, got %T", value)
	}

	switch HospitalStatus(str) {
	case HospitalActive, HospitalDisabled:
		*s = HospitalStatus(str)
		return nil
	default:
		return fmt.Errorf("invalid HospitalStatus value: %s", str)
	}
}
