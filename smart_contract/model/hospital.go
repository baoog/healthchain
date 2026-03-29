package model

import (
	"crypto/sha256"
	"encoding/hex"
)

type HospitalStatus string

const (
	KeyHospital = "HOSPITAL_%s"
)

const (
	HospitalActive   HospitalStatus = "ACTIVE"
	HospitalDisabled HospitalStatus = "DISABLED"
)

type Hospital struct {
	HospitalID string         `json:"hospital_id"` // 64-char, deterministic
	Name       string         `json:"name"`
	Status     HospitalStatus `json:"status"`
	CreatedAt  string         `json:"created_at"` // RFC3339
}

func GenerateHospitalID(seed string) string {
	sum := sha256.Sum256([]byte("HOSPITAL_SEED:" + seed))
	hashHex := hex.EncodeToString(sum[:]) // 64 chars
	return "HOSPITAL_" + hashHex[:55]
}
