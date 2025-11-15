package vault

import (
	"consentcc/config"
	"context"
	"encoding/base64"
	"fmt"

	vapi "github.com/hashicorp/vault/api"
)

type Client struct {
	api *vapi.Client
}

type IVaultClient interface {
	Encrypt(keyName string, b64Plain string) (string, error)
	StoreBundleKey(ctx context.Context, bundleId, path string, key []byte) error
	GetBundleKey(ctx context.Context, bundleId, path string) ([]byte, error)
}

func NewClient(cfg *config.Config) (*Client, error) {
	c, err := vapi.NewClient(&vapi.Config{Address: cfg.VaultConfig.VaultAddr})
	if err != nil {
		return nil, err
	}
	c.SetToken(cfg.VaultConfig.VaultToken)
	return &Client{api: c}, nil
}

// Example: wrap a DEK using transit
func (v *Client) Encrypt(keyName string, b64Plain string) (string, error) {
	path := fmt.Sprintf("transit/encrypt/%s", keyName)
	resp, err := v.api.Logical().Write(path, map[string]any{"plaintext": b64Plain})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Data == nil {
		return "", fmt.Errorf("no data")
	}
	ct, _ := resp.Data["ciphertext"].(string)
	return ct, nil
}

func (k *Client) StoreBundleKey(ctx context.Context, bundleId, path string, key []byte) error {
	data := map[string]interface{}{
		"k": base64.StdEncoding.EncodeToString(key),
	}

	secretPath := fmt.Sprintf("%s/%s", path, bundleId) // secret/ehr/keys/<bundleId>
	_, err := k.api.KVv2("secret").Put(ctx, secretPath, data)
	if err != nil {
		return fmt.Errorf("vault put bundle key: %w", err)
	}
	return nil
}

func (k *Client) GetBundleKey(ctx context.Context, bundleId, path string) ([]byte, error) {
	secretPath := fmt.Sprintf("%s/%s", path, bundleId)
	secret, err := k.api.KVv2("secret").Get(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("vault get bundle key: %w", err)
	}

	raw, ok := secret.Data["k"].(string)
	if !ok || raw == "" {
		return nil, fmt.Errorf("vault: key missing for bundle %s", bundleId)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("vault: decode key: %w", err)
	}
	return key, nil
}
