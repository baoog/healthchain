package jwt

import (
	"consentcc/config"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type (
	JWTClientProvider interface {
		Sign(userID, role, nationalIdHash string) (string, error)
		Verify(tokenStr string) (*Claims, error)
	}

	JWTClient struct {
		// Add necessary fields for JWT client
		SigningKey []byte
		ttl        time.Duration
	}
	Claims struct {
		UserID         string `json:"userId"`
		Role           string `json:"role"`
		NationalIdHash string `json:"nationalIdHash"`
		Department     string `json:"department,omitempty"`
		jwt.RegisteredClaims
	}
)

func NewJWTClient(cfg *config.Config) *JWTClient {
	return &JWTClient{
		SigningKey: []byte(cfg.JWTConfig.SigningKey),
		ttl:        cfg.JWTConfig.TTL,
	}
}

func (j *JWTClient) Sign(userID, role, nationalIdHash string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:         userID,
		Role:           role,
		NationalIdHash: nationalIdHash,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.SigningKey)
}

func (j *JWTClient) Verify(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return j.SigningKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
