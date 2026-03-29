package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateHospitalSeed generates a cryptographically secure random seed
// Length: 32 bytes → 64 hex chars
func GenerateHospitalSeed() (string, error) {
	b := make([]byte, 32) // 256-bit
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate hospital seed: %w", err)
	}
	return hex.EncodeToString(b), nil
}
