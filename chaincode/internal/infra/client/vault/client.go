package vault

import (
	"consentcc/config"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	vapi "github.com/hashicorp/vault/api"
)

type Client struct {
	api              *vapi.Client
	privateKeyPath   string
	bundlePrivateKey string
	hospitalIDPath   string
	hospitalID       string
}

type IVaultClient interface {
	Encrypt(keyName string, b64Plain string) (string, error)
	StoreBundleKey(ctx context.Context, bundleId, path string, key []byte) error
	GetBundleKey(ctx context.Context, bundleId, path string) ([]byte, error)
	ReadPrivateKeyPEM(ctx context.Context) (pem string, keyType string, keyVersion string, err error)
	ReadHospitalID(ctx context.Context) (id string, err error)
	GetTransitPublicKeyForPayload(ctx context.Context) (string, int, error)
	GetTransitPublicKeyForBundle(ctx context.Context) (string, int, error)
	SignWithTransitForPayload(
		ctx context.Context,
		hash []byte,
	) (string, error)
	SignWithTransitForBundle(
		ctx context.Context,
		hash []byte,
	) (string, error)
}

func NewClient(cfg *config.Config) (*Client, error) {
	c, err := vapi.NewClient(&vapi.Config{Address: cfg.VaultConfig.VaultAddr})
	if err != nil {
		return nil, err
	}
	c.SetToken(cfg.VaultConfig.VaultToken)
	return &Client{api: c, privateKeyPath: cfg.VaultConfig.PrivateKeyPath, hospitalIDPath: cfg.VaultConfig.HospitalIDPath, bundlePrivateKey: cfg.VaultConfig.BundlePrivateKey}, nil
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

func (v *Client) ReadPrivateKeyPEM(ctx context.Context) (pem string, keyType string, keyVersion string, err error) {
	// KV v2 path:
	// secret/data/federation/hospitals/<hospitalID>/keys/
	secretPath := fmt.Sprintf(
		v.privateKeyPath,
	)

	secret, err := v.api.KVv2("secret").Get(ctx, secretPath)
	if err != nil {
		return "", "", "", fmt.Errorf("vault read failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", "", "", fmt.Errorf("vault secret not found")
	}

	data := secret.Data

	pemAny, ok := secret.Data["private_key"]
	if !ok {
		return "", "", "", fmt.Errorf("missing private_key_pem")
	}
	pem, _ = pemAny.(string)

	if kt, _ := data["key_type"].(string); kt != "" {
		keyType = kt
	} else {
		keyType = "RSA"
	}
	if kv, _ := data["key_version"].(string); kv != "" {
		keyVersion = kv
	} else {
		keyVersion = "1"
	}

	if pem == "" {
		return "", "", "", fmt.Errorf("private_key_pem empty")
	}
	return pem, keyType, keyVersion, nil
}

func (v *Client) ReadHospitalID(ctx context.Context) (id string, err error) {
	// KV v2 path:
	// secret/data/federation/hospitals/<hospitalID>/keys/
	secretPath := v.hospitalIDPath

	secret, err := v.api.KVv2("secret").Get(ctx, secretPath)
	if err != nil {
		return "", fmt.Errorf("vault read failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("vault secret not found for hospital")
	}

	id, ok := secret.Data["id"].(string)
	if !ok {
		return "", fmt.Errorf("missing hospital id")
	}

	return id, nil
}

func (v *Client) getTransitPublicKey(ctx context.Context, keyName string) (string, int, error) {
	secret, err := v.api.Logical().ReadWithContext(ctx, keyName)
	if err != nil {
		return "", 0, err
	}
	if secret == nil || secret.Data == nil {
		return "", 0, fmt.Errorf("no data returned from vault")
	}

	// latest_version
	latestRaw, ok := secret.Data["latest_version"]
	if !ok {
		return "", 0, fmt.Errorf("latest_version not found")
	}

	latestVersionInt64, err := latestRaw.(json.Number).Int64()
	if err != nil {
		return "", 0, err
	}
	latestVersion := int(latestVersionInt64)

	// keys map
	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return "", 0, fmt.Errorf("keys map not found")
	}

	keysMap := keysRaw.(map[string]interface{})

	versionKey := fmt.Sprintf("%d", latestVersion)

	versionDataRaw, ok := keysMap[versionKey]
	if !ok {
		return "", 0, fmt.Errorf("version %s not found", versionKey)
	}

	versionData := versionDataRaw.(map[string]interface{})

	pubRaw, ok := versionData["public_key"]
	if !ok {
		return "", 0, fmt.Errorf("public_key not found")
	}

	publicKey := pubRaw.(string)

	return publicKey, latestVersion, nil
}

func (v *Client) GetTransitPublicKeyForPayload(ctx context.Context) (string, int, error) {
	path := fmt.Sprintf("transit/keys/%s", v.privateKeyPath)
	return v.getTransitPublicKey(ctx, path)
}

func (v *Client) GetTransitPublicKeyForBundle(ctx context.Context) (string, int, error) {
	path := fmt.Sprintf("transit/keys/%s", v.bundlePrivateKey)
	return v.getTransitPublicKey(ctx, path)
}

func (v *Client) SignWithTransitForPayload(
	ctx context.Context,
	hash []byte,
) (string, error) {
	keyName := fmt.Sprintf("transit/sign/%s", v.privateKeyPath)
	return v.signWithTransit(ctx, hash, keyName)
}

func (v *Client) SignWithTransitForBundle(
	ctx context.Context,
	hash []byte,
) (string, error) {
	keyName := fmt.Sprintf("transit/sign/%s", v.bundlePrivateKey)
	return v.signWithTransit(ctx, hash, keyName)
}

func (v *Client) signWithTransit(ctx context.Context, hash []byte, keyName string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(hash)
	secret, err := v.api.Logical().WriteWithContext(
		ctx,
		keyName,
		map[string]interface{}{
			"input": b64,
		},
	)
	if err != nil {
		return "", err
	}

	log.Default().Println(secret.Data)

	sigRaw, _ := secret.Data["signature"].(string)

	log.Default().Println(sigRaw)
	return sigRaw, nil
}

func (v *Client) VerifySignatureOfBundle(
	ctx context.Context,
	hash []byte,
	signature string,
) (bool, error) {
	fmt.Println("Key:", v.bundlePrivateKey)
	return v.verifyWithTransit(ctx, hash, signature, v.bundlePrivateKey)
}

func (v *Client) verifyWithTransit(ctx context.Context, hash []byte, signature, keyName string) (bool, error) {
	b64Hash := base64.StdEncoding.EncodeToString(hash)

	secret, err := v.api.Logical().WriteWithContext(
		ctx,
		fmt.Sprintf("transit/verify/%s", keyName),
		map[string]interface{}{
			"input":     b64Hash,
			"signature": signature,
		},
	)
	if err != nil {
		return false, err
	}

	valid, ok := secret.Data["valid"].(bool)
	if !ok {
		return false, fmt.Errorf("invalid response from vault")
	}

	return valid, nil
}
