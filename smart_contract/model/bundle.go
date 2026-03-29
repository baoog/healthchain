package model

type BundleType string

const (
	BundleTypeAllergy          BundleType = "ALLERGY"
	BundleTypeMedication       BundleType = "MEDICATION"
	BundleTypeBloodGroup       BundleType = "BLOOD_GROUP"
	BundleTypeEmergencySummary BundleType = "EMERGENCY_SUMMARY"
)

var ListBundleType = []BundleType{
	BundleTypeAllergy,
	BundleTypeMedication,
	BundleTypeBloodGroup,
	BundleTypeEmergencySummary,
}

type Bundle struct {
	ID          string `json:"id"`
	Owner       string `json:"owner_hash"`
	Type        string `json:"type"`
	URI         string `json:"uri"`
	SHA256      string `json:"sha256"`
	Size        string `json:"size"`
	ArchetypeID string `json:"archetype_id"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	HospitalId  string `json:"hospital_id"`
	//Checksum    string `json:"checksum"`
	Signature string `json:"signature"`
}

type ListBundleResponseWithNextToken struct {
	Bundles   []*Bundle `json:"bundles"`
	NextToken string    `json:"next_token"`
}

func IsValidBundleType(t BundleType) bool {
	for _, bt := range ListBundleType {
		if bt == t {
			return true
		}
	}
	return false
}
