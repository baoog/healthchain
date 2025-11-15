package entity

type CreateBundleReq struct {
	ID          string `json:"id,omitempty"`
	Owner       string `json:"-"`
	Type        string `json:"type,omitempty"`
	URI         string `json:"uri,omitempty"`
	SHA256      string `json:"sha_256,omitempty"`
	Size        string `json:"size,omitempty"`
	ArchetypeID string `json:"archetype_id,omitempty"`
}

type Bundle struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"` // patientId
	Type        string `json:"type"`
	URI         string `json:"uri"`
	SHA256      string `json:"sha256"`
	Size        string `json:"size"`
	ArchetypeID string `json:"archetypeId"`
}

type ConsentWithBundle struct {
	Consent *ConsentGrant `json:"consent"`
	Bundle  *Bundle       `json:"bundle"`
}
