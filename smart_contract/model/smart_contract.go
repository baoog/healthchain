package model

//type Consent struct {
//	BundleID  string `json:"bundle_id"`
//	DoctorID  string `json:"grantee_hash"`
//	Purpose   string `json:"purpose"`
//	ExpiresAt string `json:"expires_at"`
//	Status    string `json:"status"` // GRANTED/REVOKED
//	UpdatedAt string `json:"updated_at"`
//}

type ConsentWithBundle struct {
	Consent Consent `json:"consent,omitempty"`
	Bundle  *Bundle `json:"bundle,omitempty"`
}

type Consent map[string]string

func (c *Consent) ensure() {
	if c == nil {
		c = new(Consent)
	}

	if *c == nil {
		*c = make(map[string]string)
	}
}
func (c *Consent) SetBundleID(val string) {
	c.ensure()
	(*c)["bundle_id"] = val
}

func (c Consent) GetBundleID() string {
	if val, ok := c["bundle_id"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetGranteeHash(val string) {
	c.ensure()
	(*c)["grantee_hash"] = val
}

func (c Consent) GetGranteeHash() string {
	if val, ok := c["grantee_hash"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetPurpose(val string) {
	c.ensure()
	(*c)["purpose"] = val
}

func (c Consent) GetPurpose() string {
	if val, ok := c["purpose"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetStatus(val string) {
	c.ensure()
	(*c)["status"] = val
}

func (c Consent) GetStatus() string {
	if val, ok := c["status"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetExpiresTime(val string) {
	c.ensure()
	(*c)["expires_time"] = val
}

func (c Consent) GetExpiresAt() string {
	if val, ok := c["expires_at"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetUpdatedAt(val string) {
	c.ensure()
	(*c)["updated_at"] = val
}

func (c Consent) GetUpdatedAt() string {
	if val, ok := c["updated_at"]; ok {
		return val
	}

	return ""
}

func (c *Consent) SetCreatedAt(val string) {
	c.ensure()
	(*c)["created_at"] = val
}

func (c Consent) GetCreatedAt() string {
	if val, ok := c["created_at"]; ok {
		return val
	}

	return ""
}
