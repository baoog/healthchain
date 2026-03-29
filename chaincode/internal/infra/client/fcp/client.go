package fcp

import (
	"bytes"
	"consentcc/config"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	InterfaceGetBundleDetail FederationInterfaceName = "getBundleDetail"
	InterfaceGetUserInfo     FederationInterfaceName = "getUserInfo"
	InterfaceCheckWhitelist  FederationInterfaceName = "checkWhiteList"
)

type FederationInterfaceName string

type Client interface {
	RegisterHospital(ctx context.Context, req RegisterHospitalRequest) (*RegisterHospitalResponse, error)
	ResolveHospital(ctx context.Context, hospitalID string, interfaceName FederationInterfaceName) (*ResolveHospitalResponse, error)
}

type RegisterHospitalRequest struct {
	HospitalID          string                   `json:"hospital_id"`
	Name                string                   `json:"name"`
	BaseURL             string                   `json:"base_url"`
	BaseURLVersion      string                   `json:"base_url_version,omitempty"` // "v1" | "v2", default "v1"
	FederationInterface []FederationInterfaceDTO `json:"federation_interface"`
	PublicKey           string                   `json:"public_key"`
	KeyType             string                   `json:"key_type"` // "RSA" | "EC"
	ListHODID           []string                 `json:"list_hod_id,omitempty"`
}

type FederationInterfaceDTO struct {
	Name    FederationInterfaceName `json:"name"`
	Version string                  `json:"version"`
	Method  string                  `json:"method"`
	Path    string                  `json:"path"`
	Type    string                  `json:"type"`

	Auth       *AuthSchema       `json:"auth"`
	Encryption *EncryptionSchema `json:"encryption"`

	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response"`
}

type AuthSchema struct {
	Type   string `json:"type"`   // FEDERATION_TOKEN
	Header string `json:"header"` // Authorization
	Scheme string `json:"scheme"` // Bearer
}

type EncryptionSchema struct {
	RequestEncryption  string `json:"request_encryption"`  // NONE
	ResponseEncryption string `json:"response_encryption"` // JWE
	Alg                string `json:"alg"`                 // RSA-OAEP-256
	Enc                string `json:"enc"`                 // A256GCM
}

type RegisterHospitalResponse struct {
	Ok     bool `json:"ok"`
	Result *struct {
		HospitalID string `json:"hospital_id"`
		Status     string `json:"status"`
		PublicKey  string `json:"public_key"`
		KeyType    string `json:"key_type"`
		KeyVersion int    `json:"key_version"`
	} `json:"result"`
}

type ResolveHospitalResponse struct {
	HospitalID string `json:"hospital_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`

	Endpoint     EndpointResp            `json:"endpoint"`
	Key          KeyResp                 `json:"key"`
	Interface    *FederationInterfaceDTO `json:"interface"`
	ClientSecret string                  `json:"client_secret,omitempty"`
}

type KeyResp struct {
	PublicKey  string `json:"public_key"`
	KeyType    string `json:"key_type"`
	KeyVersion int    `json:"key_version"`
}

type EndpointResp struct {
	BaseURL    string `json:"base_url"`
	APIVersion string `json:"api_version"`
}

type httpClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type GetBundleDetailRequest struct {
	FieldBundleID             string `json:"BundleId"`
	FieldRequesterHospitalID  string `json:"RequesterHospitalID"`
	FieldRequesterDoctorHash  string `json:"RequesterDoctorHash"`
	FieldRequesterPatientHash string `json:"RequesterPatientHash"`

	FieldPurpose string `json:"Purpose,omitempty"`
}

func NewClient(cfg *config.Config) Client {
	return &httpClient{
		baseURL: cfg.FCPConfig.BaseURL,
		apiKey:  cfg.FCPConfig.APIKey,
		client: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

func (c *httpClient) doRequest(
	ctx context.Context,
	method, path string,
	body any,
	out any,
) error {

	var buf *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+path,
		buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("X-Admin-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var mapResp interface{}
		json.NewDecoder(resp.Body).Decode(&mapResp)
		return fmt.Errorf("fcp error: %s:%v", resp.Status, mapResp)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *httpClient) RegisterHospital(
	ctx context.Context,
	req RegisterHospitalRequest,
) (*RegisterHospitalResponse, error) {

	var resp RegisterHospitalResponse

	err := c.doRequest(
		ctx,
		http.MethodPost,
		"/fcp/v1/hospitals/register",
		req,
		&resp,
	)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *httpClient) ResolveHospital(
	ctx context.Context,
	hospitalID string,
	interfaceName FederationInterfaceName,
) (*ResolveHospitalResponse, error) {
	var mapReq = map[string]FederationInterfaceName{
		"interface_name": interfaceName,
	}

	type httpResp struct {
		OK     bool                     `json:"ok"`
		Result *ResolveHospitalResponse `json:"result,omitempty"`
	}

	var resp httpResp
	err := c.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/fcp/v1/hospitals/%s/resolve", hospitalID),
		mapReq,
		&resp,
	)
	if err != nil {
		return nil, err
	}

	if resp.Result == nil && resp.Result.Interface == nil {
		return nil, fmt.Errorf("resp.Interface is nil")
	}

	return resp.Result, nil
}
