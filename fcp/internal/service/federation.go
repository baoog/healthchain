package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fcp/internal/infra/client/fabric"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"strings"
	"time"

	"fcp/internal/adapter/entity"
	"fcp/internal/adapter/repository"
)

type FederationService interface {
	Register(ctx context.Context, req RegisterReq) (*RegisterResp, error)
	Resolve(ctx context.Context, hospitalID string, interfaceName entity.FederationInterfaceName) (*ResolveResp, error)
	RotateKey(ctx context.Context, hospitalID string, newKey string, keyType string) (*ResolveResp, error)
	SetStatus(ctx context.Context, hospitalID string, status entity.HospitalStatus) (*ResolveResp, error)
	VerifyClientSecret(ctx context.Context, hospitalID, clientID, clientSecret string) (bool, error)
}

type federationService struct {
	hospitalRepo  repository.HospitalRepository
	endpointRepo  repository.EndpointRepository
	keyRepo       repository.KeyRepository
	interfaceRepo repository.FederationInterfaceRepository
	fabricClient  fabric.FabricClient
}

func NewFederationService(
	h repository.HospitalRepository,
	e repository.EndpointRepository,
	k repository.KeyRepository,
	i repository.FederationInterfaceRepository,
	fabricClient fabric.FabricClient,
) FederationService {
	return &federationService{
		hospitalRepo:  h,
		endpointRepo:  e,
		keyRepo:       k,
		interfaceRepo: i,
		fabricClient:  fabricClient,
	}
}

type RegisterReq struct {
	HospitalID          string                           `json:"hospital_id"`
	Name                string                           `json:"name"`
	BaseURL             string                           `json:"base_url"`
	BaseURLVersion      string                           `json:"base_url_version,omitempty"` // "v1" | "v2", default "v1"
	FederationInterface []*entity.FederationInterfaceDTO `json:"federation_interface"`
	PublicKey           string                           `json:"public_key"`
	KeyType             string                           `json:"key_type"` // "RSA" | "EC"
	ListHODID           []string                         `json:"list_hod_id,omitempty"`
}

type RegisterResp struct {
	HospitalID    string                         `json:"hospital_id"`
	Endpoint      string                         `json:"endpoint"`
	PublicKey     string                         `json:"public_key"`
	KeyType       string                         `json:"key_type"`
	KeyVersion    int                            `json:"key_version"`
	Status        string                         `json:"status"`
	ClientID      string                         `json:"client_id"`
	ClientSecret  string                         `json:"client_secret"` // returned ONCE
	InterfaceName entity.FederationInterfaceName `json:"interface_name,omitempty"`
}

type ResolveResp struct {
	HospitalID string `json:"hospital_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`

	Endpoint     EndpointResp                         `json:"endpoint"`
	Key          KeyResp                              `json:"key"`
	Interface    *entity.FederationInterfacePublicDTO `json:"interface"`
	ClientSecret string                               `json:"client_secret,omitempty"`
}

type EndpointResp struct {
	BaseURL    string `json:"base_url"`
	APIVersion string `json:"api_version"`
}

type KeyResp struct {
	PublicKey  string `json:"public_key"`
	KeyType    string `json:"key_type"`
	KeyVersion int    `json:"key_version"`
}

func (s *federationService) Register(ctx context.Context, req RegisterReq) (*RegisterResp, error) {
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.PublicKey = strings.TrimSpace(req.PublicKey)
	req.KeyType = strings.TrimSpace(req.KeyType)

	if req.HospitalID == "" || req.BaseURL == "" || req.PublicKey == "" || req.KeyType == "" {
		return nil, errors.New("hospital_id, endpoint, public_key, key_type are required")
	}

	var requiredInterface = map[entity.FederationInterfaceName]struct{}{
		entity.InterfaceGetBundleDetail: struct{}{},
		entity.InterfaceCheckWhitelist:  struct{}{},
	}
	if req.FederationInterface == nil && len(req.FederationInterface) == 0 {
		return nil, errors.New("federation_interface is required")
	}

	see := 0
	for _, reqInterface := range req.FederationInterface {

		if reqInterface.Name != entity.InterfaceGetBundleDetail &&
			reqInterface.Name != entity.InterfaceGetUserInfo && reqInterface.Name != entity.InterfaceCheckWhitelist {
			return nil, errors.New("unsupported federation_interface name")
		}

		if reqInterface.Name != entity.InterfaceCheckWhitelist {
			if reqInterface.Path == "" || reqInterface.Method == "" ||
				reqInterface.Version == "" || reqInterface.Auth == nil || reqInterface.Encryption == nil ||
				reqInterface.Request == nil || reqInterface.Response == nil {
				return nil, errors.New("incomplete federation_interface details")
			}
		} else {
			if reqInterface.Path == "" || reqInterface.Method == "" ||
				reqInterface.Version == "" || reqInterface.Auth == nil {
				return nil, errors.New("incomplete federation_interface details")
			}
		}

		if _, ok := requiredInterface[reqInterface.Name]; ok {
			see++
		}
	}

	if see < 2 {
		return nil, fmt.Errorf("%s and %s are required",
			entity.InterfaceGetBundleDetail, entity.InterfaceCheckWhitelist)
	}

	//generate client credentials
	clientID := randomHex(16)
	clientSecret := randomHex(32)

	secretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash client secret: %w", err)
	}

	var hospital *entity.Hospital
	var keyVersion int

	err = s.hospitalRepo.WithTx(ctx, func(txCtx context.Context) error {
		convertListID := strings.Join(req.ListHODID, ",")
		createdAt := time.Now().UTC()
		resp, err := s.fabricClient.Contract().SubmitTransaction("RegisterHospital", req.HospitalID, req.Name, createdAt.Format(time.RFC3339), convertListID)
		if err != nil {
			return err
		}

		var hospitalResp entity.HospitalFabric
		err = json.Unmarshal(resp, &hospitalResp)
		if err != nil {
			return err
		}

		// 1. create hospital
		h := &entity.Hospital{
			HospitalID:       hospitalResp.HospitalID,
			Name:             req.Name,
			Status:           entity.HospitalActive,
			ClientID:         clientID,
			ClientSecretHash: string(secretHash),
		}

		if err := s.hospitalRepo.Create(txCtx, h); err != nil {
			return err
		}
		hospital = h

		// 2. create endpoint
		if err := s.endpointRepo.CreateEndpoint(txCtx, &entity.HospitalEndpoint{
			HospitalID: h.ID,
			APIBaseURL: req.BaseURL,
			APIVersion: req.BaseURLVersion,
			CreatedAt:  time.Now().UTC(),
			IsPrimary:  true,
		}); err != nil {
			return err
		}

		// 3. create public key (version 1)
		key := &entity.HospitalKey{
			HospitalID: h.ID,
			PublicKey:  req.PublicKey,
			KeyType:    req.KeyType,
			Version:    1,
			IsActive:   true,
		}
		if err := s.keyRepo.AddKeyHistory(txCtx, key); err != nil {
			return err
		}
		keyVersion = key.Version

		// 4. create federation interface record (if provided)
		var listFederationInterface = make([]*entity.FederationInterface, 0)

		for _, r := range req.FederationInterface {
			fdrInterface, err := r.ToEntity(h.ID)
			if err != nil {
				return err
			}
			listFederationInterface = append(listFederationInterface, fdrInterface)
		}

		if err := s.interfaceRepo.CreateInterFace(txCtx, listFederationInterface); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &RegisterResp{
		HospitalID:   hospital.HospitalID,
		Endpoint:     req.BaseURL,
		PublicKey:    req.PublicKey,
		KeyType:      req.KeyType,
		KeyVersion:   keyVersion,
		Status:       string(hospital.Status),
		ClientID:     clientID,
		ClientSecret: clientSecret, // returned ONCE
	}, nil
}

func (s *federationService) Resolve(
	ctx context.Context,
	hospitalID string,
	interfaceName entity.FederationInterfaceName,
) (*ResolveResp, error) {

	hospitalID = strings.TrimSpace(hospitalID)
	if hospitalID == "" {
		return nil, errors.New("hospital_id is required")
	}

	log.Default().Println("Resolve")

	// 1. Load hospital
	h, err := s.hospitalRepo.GetByHospitalID(ctx, hospitalID)
	if err != nil {
		return nil, err
	}
	if h.Status != entity.HospitalActive {
		return nil, fmt.Errorf("hospital %s is %s", hospitalID, h.Status)
	}

	log.Default().Println("Get Hospital", h.HospitalID, h.Status)

	// 2. Load primary endpoint
	ep, err := s.endpointRepo.FindPrimaryByHospitalID(ctx, h.ID)
	if err != nil {
		return nil, fmt.Errorf("primary endpoint not found")
	}

	log.Default().Println("Get Endpoint", ep.APIBaseURL, ep.IsPrimary)

	// 3. Load active public key
	key, err := s.keyRepo.FindActiveByHospitalID(ctx, h.ID)
	if err != nil {
		return nil, fmt.Errorf("active public key not found")
	}

	log.Default().Println("FindActiveByHospitalID", key)

	// 4. Load ONLY requested interface
	iface, err := s.interfaceRepo.FindByHospitalAndName(ctx, h.ID, interfaceName)
	if err != nil {
		log.Default().Println("FindByHospitalAndName error", err)
		return nil, fmt.Errorf("interface %s not supported by hospital", interfaceName)
	}

	log.Default().Println("FindByHospitalAndName", iface)

	// 5. Build response
	return &ResolveResp{
		HospitalID:   h.HospitalID,
		Name:         h.Name,
		Status:       string(h.Status),
		ClientSecret: h.ClientSecretHash,

		Endpoint: EndpointResp{
			BaseURL:    ep.APIBaseURL,
			APIVersion: ep.APIVersion,
		},

		Key: KeyResp{
			PublicKey:  key.PublicKey,
			KeyType:    key.KeyType,
			KeyVersion: key.Version,
		},

		Interface: iface.ToPublicDTO(),
	}, nil
}

func (s *federationService) RotateKey(ctx context.Context, hospitalID string, newKey string, keyType string) (*ResolveResp, error) {
	hospitalID = strings.TrimSpace(hospitalID)
	if hospitalID == "" {
		return nil, errors.New("hospital_id is required")
	}
	newKey = strings.TrimSpace(newKey)
	keyType = strings.TrimSpace(keyType)
	if newKey == "" || keyType == "" {
		return nil, errors.New("public_key and key_type are required")
	}

	h, err := s.hospitalRepo.GetByHospitalID(ctx, hospitalID)
	if err != nil {
		return nil, err
	}

	// rotate in key history (deactivate old active key, add new version)
	err = s.hospitalRepo.WithTx(ctx, func(txCtx context.Context) error {
		active, err := s.keyRepo.FindActiveByHospitalID(txCtx, h.ID)
		if err == nil && active != nil {
			// best-effort deactivate (ignore if repo doesn't support explicit update)
			active.IsActive = false
			_ = s.keyRepo.UpdateKey(txCtx, active)
			nextVer := active.Version + 1
			return s.keyRepo.AddKeyHistory(txCtx, &entity.HospitalKey{
				HospitalID: h.ID,
				PublicKey:  newKey,
				KeyType:    keyType,
				Version:    nextVer,
				IsActive:   true,
			})
		}

		// if no active key exists, start at version 1
		return s.keyRepo.AddKeyHistory(txCtx, &entity.HospitalKey{
			HospitalID: h.ID,
			PublicKey:  newKey,
			KeyType:    keyType,
			Version:    1,
			IsActive:   true,
		})
	})
	if err != nil {
	}

	return &ResolveResp{
		HospitalID: h.HospitalID,
		Name:       h.Name,
		Status:     string(h.Status),
		Endpoint:   EndpointResp{},
		Key:        KeyResp{PublicKey: newKey, KeyType: keyType},
		Interface:  nil,
	}, nil
}

func (s *federationService) SetStatus(ctx context.Context, hospitalID string, status entity.HospitalStatus) (*ResolveResp, error) {
	hospitalID = strings.TrimSpace(hospitalID)
	if hospitalID == "" {
		return nil, errors.New("hospital_id is required")
	}

	h, err := s.hospitalRepo.GetByHospitalID(ctx, hospitalID)
	if err != nil {
		return nil, err
	}

	h.Status = status
	if err := s.hospitalRepo.Update(ctx, h); err != nil {
		return nil, err
	}
	// Resolve() blocks disabled; so return struct directly
	return &ResolveResp{
		HospitalID: h.HospitalID,
		Name:       h.Name,
		Status:     string(h.Status),
	}, nil
}

func (s *federationService) VerifyClientSecret(ctx context.Context, hospitalID, clientID, clientSecret string) (bool, error) {
	h, err := s.hospitalRepo.GetByHospitalID(ctx, hospitalID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if h.ClientID != clientID {
		return false, nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h.ClientSecretHash), []byte(clientSecret)); err != nil {
		return false, nil
	}
	return true, nil
}

func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
