package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

type BundleCipher struct {
	Nonce      []byte
	Ciphertext []byte
	SHA256Hex  string
	Size       int64
}

// GenerateDataKey returns a new 32-byte AES key.
func GenerateDataKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate data key: %w", err)
	}
	return key, nil
}

// EncryptBundle encrypts payload with AES-256-GCM and computes SHA256 of (nonce||ciphertext).
func EncryptBundle(payload []byte, key []byte) (*BundleCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, payload, nil)

	combined := make([]byte, len(nonce)+len(ciphertext))
	copy(combined, nonce)
	copy(combined[len(nonce):], ciphertext)

	sum := sha256.Sum256(combined)

	return &BundleCipher{
		Nonce:      nonce,
		Ciphertext: ciphertext,
		SHA256Hex:  hex.EncodeToString(sum[:]),
		Size:       int64(len(combined)),
	}, nil
}

// BuildReader builds an io.Reader for uploading to MinIO.
func (b *BundleCipher) BuildReader() io.Reader {
	buf := bytes.NewBuffer(nil)
	buf.Grow(len(b.Nonce) + len(b.Ciphertext))
	buf.Write(b.Nonce)
	buf.Write(b.Ciphertext)
	return buf
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func DecryptAESGCM(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return gcm.Open(nil, nonce, ct, nil)
}

func HashNationalID(countryCode, nationalID string) string {
	raw := fmt.Sprintf("%s:%s", countryCode, nationalID)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func HashFabricIdentity(userID string) string {
	raw := fmt.Sprintf("%s", userID)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
