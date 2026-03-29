package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	InterfaceGetBundleDetail FederationInterfaceName = "getBundleDetail"
	InterfaceGetUserInfo     FederationInterfaceName = "getUserInfo"
	InterfaceCheckWhitelist  FederationInterfaceName = "checkWhiteList"
)

type JSONB map[string]any
type FederationInterface struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	HospitalID uuid.UUID `gorm:"type:uuid;index;not null"`

	Type    string                  `gorm:"type:varchar;not null"`
	Name    FederationInterfaceName `gorm:"size:64;not null"`
	Version string                  `gorm:"size:16;not null"`

	Method string `gorm:"size:8;not null"`
	Path   string `gorm:"size:128;not null"`

	Auth AuthSchema `gorm:"embedded;embeddedPrefix:auth_"`

	Encryption EncryptionSchema `gorm:"embedded;embeddedPrefix:enc_"`

	// Typed schemas
	RequestSchema  JSONB `gorm:"type:jsonb;not null"`
	ResponseSchema JSONB `gorm:"type:jsonb;not null"`

	CreatedAt time.Time
}

func (FederationInterface) TableName() string {
	return "federation_interfaces"
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

type GetBundleDetailRequest struct {
	BundleID            string `json:"BundleId"`
	RequesterHospitalID string `json:"RequesterHospitalID"`
	RequesterDoctorHash string `json:"RequesterDoctorHash"`
	Purpose             string `json:"Purpose,omitempty"`
}

func (r *GetBundleDetailRequest) ToMap() (map[string]any, error) {
	if r == nil {
		return nil, nil
	}

	var result map[string]any
	bytes, _ := json.Marshal(*r)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, fmt.Errorf("GetUserInfoResponse to map: %w", err)
	}
	return result, nil
}

func (r *GetBundleDetailRequest) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(r)
	return string(bytes), nil
}

func (r *GetBundleDetailRequest) Scan(src interface{}) error {
	bytes, _ := json.Marshal(src)
	err := json.Unmarshal(bytes, r)
	if err != nil {
		return fmt.Errorf("GetBundleDetailRequest must be string, got %v", err.Error())
	}
	return nil
}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("JSONB: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, j)
}

type GetBundleDetailResponse struct {
	BundleID   string      `json:"BundleID"`
	BundleType string      `json:"BundleType"`
	Ciphertext string      `json:"Ciphertext"` // JWE base64
	KeyVersion int         `json:"KeyVersion"`
	ExpiresAt  string      `json:"ExpiresAt"`
	Data       interface{} `json:"Data,omitempty"`
}

func (r *GetBundleDetailResponse) ToMap() (map[string]any, error) {
	if r == nil {
		return nil, nil
	}

	var result map[string]any
	bytes, _ := json.Marshal(*r)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, fmt.Errorf("GetUserInfoResponse to map: %w", err)
	}
	return result, nil
}

func (r *GetBundleDetailResponse) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(r)
	return string(bytes), nil
}

func (r *GetBundleDetailResponse) Scan(src interface{}) error {
	bytes, _ := json.Marshal(src)
	err := json.Unmarshal(bytes, r)
	if err != nil {
		return fmt.Errorf("GetBundleDetailRequest must be string, got %v", err.Error())
	}
	return nil
}

type GetUserInfoRequest struct {
	UserHash            string `json:"UserHash"`
	RequesterHospitalID string `json:"RequesterHospitalID"`
}

func (r *GetUserInfoRequest) ToMap() (map[string]any, error) {
	if r == nil {
		return nil, nil
	}

	var result map[string]any
	bytes, _ := json.Marshal(*r)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, fmt.Errorf("GetUserInfoResponse to map: %w", err)
	}
	return result, nil
}
func (r *GetUserInfoRequest) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(r)
	return string(bytes), nil
}

func (r *GetUserInfoRequest) Scan(src interface{}) error {
	bytes, _ := json.Marshal(src)
	err := json.Unmarshal(bytes, r)
	if err != nil {
		return fmt.Errorf("GetBundleDetailRequest must be string, got %v", err.Error())
	}
	return nil
}

type GetUserInfoResponse struct {
	UserHash    string `json:"UserHash"`
	DisplayName string `json:"DisplayName"`
	Gender      string `json:"Gender"`
	DateOfBirth string `json:"DateOfBirth"`
	CountryCode string `json:"CountryCode"`
}

func (r *GetUserInfoResponse) ToMap() (map[string]any, error) {
	if r == nil {
		return nil, nil
	}

	var result map[string]any
	bytes, _ := json.Marshal(*r)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, fmt.Errorf("GetUserInfoResponse to map: %w", err)
	}
	return result, nil
}
func (r *GetUserInfoResponse) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(r)
	return string(bytes), nil
}

func (r *GetUserInfoResponse) Scan(src interface{}) error {
	bytes, _ := json.Marshal(src)
	err := json.Unmarshal(bytes, r)
	if err != nil {
		return fmt.Errorf("GetBundleDetailRequest must be string, got %v", err.Error())
	}
	return nil
}

type FederationInterfaceDTO struct {
	Name    FederationInterfaceName `json:"name"`
	Version string                  `json:"version"`
	Method  string                  `json:"method"`
	Path    string                  `json:"path"`
	Type    string                  `json:"type,omitempty"`

	Auth       *AuthSchema       `json:"auth"`
	Encryption *EncryptionSchema `json:"encryption"`

	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response"`
}

func (dto *FederationInterfaceDTO) ToEntity(hospitalID uuid.UUID) (*FederationInterface, error) {
	var mapReq map[string]any
	var mapRes map[string]any

	switch dto.Name {
	case InterfaceGetBundleDetail:
		req := &GetBundleDetailRequest{}
		if err := json.Unmarshal(dto.Request, &req); err != nil {
			return nil, fmt.Errorf("invalid request schema: %w", err)
		}

		res := &GetBundleDetailResponse{}
		err := json.Unmarshal(dto.Response, &res)
		if err != nil {
			return nil, fmt.Errorf("invalid response schema: %w", err)
		}

		mapReq, err = req.ToMap()
		if err != nil {
			return nil, err
		}

		mapRes, err = res.ToMap()
		if err != nil {
			return nil, err
		}

	case InterfaceGetUserInfo:
		req := &GetUserInfoRequest{}
		if err := json.Unmarshal(dto.Request, &req); err != nil {
			return nil, fmt.Errorf("invalid request schema: %w", err)
		}
		res := &GetUserInfoResponse{}
		err := json.Unmarshal(dto.Response, &res)
		if err != nil {
			return nil, fmt.Errorf("invalid response schema: %w", err)
		}

		mapReq, err = req.ToMap()
		if err != nil {
			return nil, err
		}

		mapRes, err = res.ToMap()
		if err != nil {
			return nil, err
		}
	case InterfaceCheckWhitelist:

	default:
		return nil, fmt.Errorf("unsupported interface: %s", dto.Name)
	}

	fdrInterface := &FederationInterface{
		HospitalID:     hospitalID,
		Name:           dto.Name,
		Version:        dto.Version,
		Method:         dto.Method,
		Path:           dto.Path,
		RequestSchema:  mapReq,
		ResponseSchema: mapRes,
		Type:           dto.Type,
	}

	if dto.Auth != nil {
		fdrInterface.Auth = *dto.Auth
	}

	if dto.Encryption != nil {
		fdrInterface.Encryption = *dto.Encryption
	}

	return fdrInterface, nil
}

func (n FederationInterfaceName) Value() (driver.Value, error) {
	switch n {
	case InterfaceGetBundleDetail, InterfaceGetUserInfo, InterfaceCheckWhitelist:
		return string(n), nil
	default:
		return nil, fmt.Errorf("invalid FederationInterfaceName: %s", n)
	}
}

func (n *FederationInterfaceName) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("FederationInterfaceName cannot be NULL")
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("FederationInterfaceName must be string, got %T", value)
	}

	switch FederationInterfaceName(str) {
	case InterfaceGetBundleDetail, InterfaceGetUserInfo, InterfaceCheckWhitelist:
		*n = FederationInterfaceName(str)
		return nil
	default:
		return fmt.Errorf("invalid FederationInterfaceName value: %s", str)
	}
}

// FederationInterfacePublicDTO is a public DTO representation of FederationInterface with typed request/response schemas.
type FederationInterfacePublicDTO struct {
	Name    FederationInterfaceName `json:"name"`
	Version string                  `json:"version"`
	Method  string                  `json:"method"`
	Path    string                  `json:"path"`

	Auth       AuthSchema       `json:"auth"`
	Encryption EncryptionSchema `json:"encryption"`

	Request  interface{} `json:"request"`
	Response interface{} `json:"response"`
}

func (fi *FederationInterface) ToPublicDTO() *FederationInterfacePublicDTO {
	return &FederationInterfacePublicDTO{
		Name:       fi.Name,
		Version:    fi.Version,
		Method:     fi.Method,
		Path:       fi.Path,
		Auth:       fi.Auth,
		Encryption: fi.Encryption,
		Request:    fi.RequestSchema,
		Response:   fi.ResponseSchema,
	}
}
