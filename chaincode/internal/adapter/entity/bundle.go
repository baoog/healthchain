package entity

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

var hospitalID = ""

type (
	BundleType  string
	ArchetypeID string
)

const (
	BundleTypeAllergy          BundleType = "ALLERGY"
	BundleTypeMedication       BundleType = "MEDICATION"
	BundleTypeBloodGroup       BundleType = "BLOOD_GROUP"
	BundleTypeEmergencySummary BundleType = "EMERGENCY_SUMMARY"

	// ALLERGY
	ArchAllergyIntolerance ArchetypeID = "openEHR-EHR-EVALUATION.allergy_intolerance.v1"
	ArchAdverseReaction    ArchetypeID = "openEHR-EHR-EVALUATION.adverse_reaction.v1"

	// MEDICATION
	ArchMedicationOrder ArchetypeID = "openEHR-EHR-INSTRUCTION.medication_order.v1"
	ArchMedicationList  ArchetypeID = "openEHR-EHR-ADMIN_ENTRY.medication_list.v1"

	// BLOOD GROUP
	ArchBloodGroup ArchetypeID = "openEHR-EHR-OBSERVATION.blood_group.v1"

	// EMERGENCY SUMMARY
	ArchEmergencySummary ArchetypeID = "openEHR-EHR-EVALUATION.emergency_summary.v1"
)

var ArchetypesByBundleType = map[BundleType][]ArchetypeID{
	BundleTypeAllergy: {
		ArchAllergyIntolerance,
		ArchAdverseReaction,
	},

	BundleTypeMedication: {
		ArchMedicationOrder,
		ArchMedicationList,
	},

	BundleTypeBloodGroup: {
		ArchBloodGroup,
	},

	BundleTypeEmergencySummary: {
		ArchEmergencySummary,
	},
}

var DefaultArchetypeByBundleType = map[BundleType]ArchetypeID{
	BundleTypeAllergy:          ArchAllergyIntolerance,
	BundleTypeMedication:       ArchMedicationOrder,
	BundleTypeBloodGroup:       ArchBloodGroup,
	BundleTypeEmergencySummary: ArchEmergencySummary,
}

type (
	CreateBundleReq struct {
		ID          string      `json:"-"`
		Type        BundleType  `json:"type,omitempty"`
		SHA256      string      `json:"sha_256,omitempty"`
		Size        string      `json:"-"`
		ArchetypeID string      `json:"archetype_id,omitempty"`
		Data        interface{} `json:"data,omitempty"`
		Owner       string      `json:"owner,omitempty"`
	}

	Bundle struct {
		ID          string      `json:"id" gorm:"column:id;primaryKey"`
		OwnerHash   string      `json:"owner_hash" gorm:"column:owner_hash;type:varchar(64);index;not null"` // patientId
		Type        BundleType  `json:"type" gorm:"column:type;type:varchar(30);not null"`
		ObjectURI   string      `json:"uri" gorm:"column:object_uri;type:text;not null"`
		SHA256      string      `json:"sha256" gorm:"column:sha256;type:varchar(64);not null"`
		Size        string      `json:"size" gorm:"column:size;type:varchar(20)"`
		ArchetypeID string      `json:"archetype_id" gorm:"type:varchar(255)"`
		Data        interface{} `json:"data,omitempty" gorm:"-:all"`
		CreatedAt   *time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
		Signature   string      `json:"signature" gorm:"-:all"`

		//UpdatedAt   *time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	}

	ConsentWithBundle struct {
		Consent *Consent `json:"consent"`
		Bundle  *Bundle  `json:"bundle"`
		Grantee *Grantee `json:"grantee,omitempty"`
	}

	// 1) Allergy
	AllergyData struct {
		Allergen  string `json:"allergen"`            // e.g. "Penicillin"
		Reaction  string `json:"reaction"`            // e.g. "Rash and swelling"
		Severity  string `json:"severity"`            // e.g. "mild", "moderate", "severe"
		OnsetDate string `json:"onsetDate,omitempty"` // "YYYY-MM-DD"
		Notes     string `json:"notes,omitempty"`
	}

	// 2) Medication
	MedicationData struct {
		DrugName  string `json:"drug_name"`
		Dose      string `json:"dose"`
		Frequency string `json:"frequency"`
		Route     string `json:"route"`
		StartDate string `json:"start_date,omitempty"`
		EndDate   string `json:"end_date,omitempty"`
		Notes     string `json:"notes,omitempty"`
	}

	// 3) Blood group
	BloodGroupData struct {
		ABO           string `json:"abo"`                      // "A", "B", "AB", "O"
		Rh            string `json:"rh"`                       // "+", "-"
		LastConfirmed string `json:"last_confirmed,omitempty"` // date
	}

	// 4) Emergency summary
	EmergencySummaryData struct {
		Problems           []string `json:"problems"`            // ["CAD", "Type 2 diabetes"]
		Allergies          []string `json:"allergies,omitempty"` // ["Penicillin"]
		CurrentMedications []string `json:"current_medications,omitempty"`
		Notes              string   `json:"notes,omitempty"`
	}

	ListBundleResponseWithNextToken struct {
		Bundles   []*BundleFromFabric `json:"bundles,omitempty"`
		NextToken string              `json:"next_token,omitempty"`
	}

	BundleFromFabric struct {
		ID          string      `json:"id"`
		Owner       string      `json:"owner_hash"`
		Type        string      `json:"type"`
		URI         string      `json:"uri"`
		SHA256      string      `json:"sha256"`
		Size        string      `json:"size"`
		ArchetypeID string      `json:"archetype_id"`
		CreatedAt   string      `json:"created_at,omitempty"`
		UpdatedAt   string      `json:"updated_at,omitempty"`
		HospitalId  string      `json:"hospital_id"`
		IsExternal  *bool       `json:"is_external,omitempty"`
		Signature   string      `json:"signature"`
		Data        interface{} `json:"data,omitempty"`
		User        *User       `json:"user,omitempty"`
	}

	BundleWithUser struct {
		ID          string      `json:"id" gorm:"column:id;primaryKey"`
		OwnerHash   string      `json:"owner_hash" gorm:"column:owner_hash;type:varchar(64);index;not null"` // patientId
		Type        BundleType  `json:"type" gorm:"column:type;type:varchar(30);not null"`
		ObjectURI   string      `json:"uri" gorm:"column:object_uri;type:text;not null"`
		SHA256      string      `json:"sha256" gorm:"column:sha256;type:varchar(64);not null"`
		Size        string      `json:"size" gorm:"column:size;type:varchar(20)"`
		ArchetypeID string      `json:"archetype_id" gorm:"type:varchar(255)"`
		Data        interface{} `json:"data,omitempty" gorm:"-:all"`
		CreatedAt   *time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
		User        *User       `json:"user,omitempty"`
		IsExternal  *bool       `json:"is_external,omitempty"`
	}

	ListBundleUsersResponseWithNextToken struct {
		Bundles   *[]BundleWithUser `json:"bundles,omitempty"`
		NextToken string            `json:"next_token,omitempty"`
	}

	SignedBundlePayload struct {
		OK bool `json:"ok"`

		Bundle struct {
			ID          string      `json:"id"`
			Owner       string      `json:"owner_hash"`
			Type        string      `json:"type"`
			SHA256      string      `json:"sha256"`
			ArchetypeID string      `json:"archetype_id"`
			HospitalID  string      `json:"hospital_id"`
			Data        interface{} `json:"data"`
		} `json:"bundle"`

		User *User `json:"user,omitempty"`
	}
)

func ToBundleType(s string) BundleType {
	switch strings.ToUpper(s) {
	case "ALLERGY":
		return BundleTypeAllergy
	case "MEDICATION":
		return BundleTypeMedication
	case "BLOOD_GROUP":
		return BundleTypeBloodGroup
	case "EMERGENCY_SUMMARY":
		return BundleTypeEmergencySummary
	default:
		return BundleType("")
	}
}

func ToListBundleType(strs []string) []BundleType {
	var types []BundleType
	for _, s := range strs {
		bt := ToBundleType(s)
		if bt != "" {
			types = append(types, bt)
		}
	}
	return types
}

func (req *CreateBundleReq) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type        BundleType      `json:"type"`
		ArchetypeID string          `json:"archetype_id,omitempty"`
		Data        json.RawMessage `json:"data"`
		Owner       string          `json:"owner"`
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("invalid CreateBundleReq: %w", err)
	}

	if raw.Type == "" {
		return fmt.Errorf("bundle type is required")
	}

	req.Type = raw.Type
	req.Owner = raw.Owner

	if raw.ArchetypeID == "" {
		def, ok := DefaultArchetypeByBundleType[raw.Type]
		if !ok {
			return fmt.Errorf("no default archetype for bundle type %s", raw.Type)
		}
		req.ArchetypeID = string(def)
	} else {
		// 🔒 Validate explicit archetype
		if !IsValidArchetypeForBundleType(raw.Type, raw.ArchetypeID) {
			return fmt.Errorf(
				"invalid archetype '%s' for bundle type '%s'",
				raw.ArchetypeID,
				raw.Type,
			)
		}
		req.ArchetypeID = raw.ArchetypeID
	}

	switch raw.Type {

	case BundleTypeAllergy:
		var d AllergyData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			return fmt.Errorf("invalid allergy data: %w", err)
		}
		if d.Allergen == "" {
			return fmt.Errorf("allergen is required")
		}
		req.Data = d

	case BundleTypeMedication:
		var d MedicationData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			return fmt.Errorf("invalid medication data: %w", err)
		}
		if d.DrugName == "" || d.Dose == "" {
			return fmt.Errorf("drug_name and dose are required")
		}
		req.Data = d

	case BundleTypeBloodGroup:
		var d BloodGroupData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			return fmt.Errorf("invalid blood_group data: %w", err)
		}
		if d.ABO == "" || d.Rh == "" {
			return fmt.Errorf("abo and rh are required")
		}
		req.Data = d

	case BundleTypeEmergencySummary:
		var d EmergencySummaryData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			return fmt.Errorf("invalid emergency_summary data: %w", err)
		}
		if len(d.Problems) == 0 {
			return fmt.Errorf("at least one problem is required")
		}
		req.Data = d

	default:
		return fmt.Errorf("unsupported bundle type: %s", raw.Type)
	}

	return nil
}

type BundleTypeArray []BundleType

// ---- Save to DB (Go → Postgres) ----
func (a BundleTypeArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	items := make([]string, len(a))
	for i, v := range a {
		items[i] = string(v)
	}
	return "{" + strings.Join(items, ",") + "}", nil
}

// ---- Load from DB (Postgres → Go) ----
func (a *BundleTypeArray) Scan(src interface{}) error {
	if src == nil {
		*a = []BundleType{}
		return nil
	}

	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into BundleTypeArray", src)
	}

	s = strings.Trim(s, "{}")
	if s == "" {
		*a = []BundleType{}
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]BundleType, len(parts))
	for i, p := range parts {
		out[i] = BundleType(strings.TrimSpace(p))
	}
	*a = out
	return nil
}

func (t BundleType) Value() (driver.Value, error) {
	return string(t), nil
}

func (t *BundleType) Scan(src interface{}) error {
	if src == nil {
		*t = ""
		return nil
	}

	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into BundleType", src)
	}

	*t = BundleType(s)
	return nil
}

func IsValidArchetypeForBundleType(
	bundleType BundleType,
	archetypeId string,
) bool {
	list, ok := ArchetypesByBundleType[bundleType]
	if !ok {
		return false
	}

	for _, a := range list {
		if string(a) == archetypeId {
			return true
		}
	}
	return false
}

func GetHospitalID() string {
	return hospitalID
}

func SetHospitalID(id string) {
	hospitalID = id
}

func (b *BundleFromFabric) ToBundle(data interface{}) *Bundle {
	createdAt, _ := time.Parse(time.RFC3339, b.CreatedAt)
	bundle := &Bundle{
		ID:          b.ID,
		OwnerHash:   b.Owner,
		Type:        ToBundleType(b.Type),
		ObjectURI:   b.URI,
		SHA256:      b.SHA256,
		Size:        b.Size,
		ArchetypeID: b.ArchetypeID,
		CreatedAt:   &createdAt,
	}

	if data != nil {
		bundle.Data = data
	}

	return bundle
}

func (b *BundleFromFabric) UnmarshalJSON(data []byte) error {
	log.Default().Println("UnmarshalJSON BundleFromFabric")
	type Alias BundleFromFabric // prevent recursion

	aux := struct {
		Data json.RawMessage `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	// First unmarshal everything normally
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// If no data, nothing to do
	if len(aux.Data) == 0 {
		return nil
	}

	// Switch by bundle type
	switch b.Type {

	case "ALLERGY":
		var d AllergyData
		if err := json.Unmarshal(aux.Data, &d); err != nil {
			return fmt.Errorf("invalid allergy data: %w", err)
		}
		b.Data = d

	case "MEDICATION":
		var d MedicationData
		if err := json.Unmarshal(aux.Data, &d); err != nil {
			return fmt.Errorf("invalid medication data: %w", err)
		}
		b.Data = d

	case "BLOOD_GROUP":
		var d BloodGroupData
		if err := json.Unmarshal(aux.Data, &d); err != nil {
			return fmt.Errorf("invalid blood group data: %w", err)
		}
		b.Data = d

	case "EMERGENCY_SUMMARY":
		var d EmergencySummaryData
		if err := json.Unmarshal(aux.Data, &d); err != nil {
			return fmt.Errorf("invalid emergency summary data: %w", err)
		}
		b.Data = d

	default:
		// fallback generic map
		log.Default().Println("fallback generic map")
		var d map[string]any
		if err := json.Unmarshal(aux.Data, &d); err != nil {
			return err
		}
		b.Data = d
	}

	return nil
}

func (p *SignedBundlePayload) UnmarshalJSON(data []byte) error {

	// Prevent recursion
	type Alias SignedBundlePayload

	// Temporary struct with raw Data
	aux := struct {
		Bundle struct {
			ID          string          `json:"id"`
			Owner       string          `json:"owner_hash"`
			Type        string          `json:"type"`
			SHA256      string          `json:"sha256"`
			ArchetypeID string          `json:"archetype_id"`
			HospitalID  string          `json:"hospital_id"`
			Data        json.RawMessage `json:"data"`
		} `json:"bundle"`

		User *User `json:"user,omitempty"`
		OK   bool  `json:"ok"`
	}{}

	// Step 1: Unmarshal outer structure
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Copy simple fields
	p.OK = aux.OK
	p.User = aux.User

	// Copy bundle meta
	p.Bundle.ID = aux.Bundle.ID
	p.Bundle.Owner = aux.Bundle.Owner
	p.Bundle.Type = aux.Bundle.Type
	p.Bundle.SHA256 = aux.Bundle.SHA256
	p.Bundle.ArchetypeID = aux.Bundle.ArchetypeID
	p.Bundle.HospitalID = aux.Bundle.HospitalID

	// Step 2: Decode Data based on Type
	if len(aux.Bundle.Data) == 0 {
		return nil
	}

	p.Bundle.Data = aux.Bundle.Data

	return nil
}
