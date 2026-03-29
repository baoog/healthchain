package model

type BreakGlassStatus string

const (
	BGRequested BreakGlassStatus = "REQUESTED"
	BGApproved  BreakGlassStatus = "APPROVED"
	BGRejected  BreakGlassStatus = "REJECTED"
	BGExpired   BreakGlassStatus = "EXPIRED"
)

type BreakGlassRequest struct {
	RequestID             string           `json:"request_id"`
	OwnerHash             string           `json:"owner_hash"`
	RequestFromHospitalID string           `json:"request_from_hospital_id"`
	DoctorHash            string           `json:"doctor_hash"`
	HODHash               string           `json:"hod_hash,omitempty"`
	Justification         string           `json:"justification"`
	AllowedTypes          []BundleType     `json:"allowed_types,omitempty"`
	ExpiresAt             string           `json:"expires_at"` // RFC3339
	Status                BreakGlassStatus `json:"status"`
	CreatedAt             string           `json:"created_at"`
	UpdatedAt             string           `json:"updated_at"`
	BundleIDs             []string         `json:"bundle_ids"`
	IsExternal            bool             `json:"is_external"`
	TotalBundles          int32            `json:"total_bundles"`
}

type ListBreakGlassRequestResponseWithNextToken struct {
	Requests  []*BreakGlassRequest `json:"requests"`
	NextToken string               `json:"next_token"`
}
