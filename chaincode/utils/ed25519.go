package utils

import (
	"bytes"
	"crypto/ed25519"
)

func GenerateEd25519PK(key string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(bytes.NewBufferString(key))
}
