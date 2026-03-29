package entity

import "time"

type AuditEvent struct {
	ID        string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"` // UUIDv4
	ActorHash string    `gorm:"column:actor_hash;type:varchar(64);index;not null"`
	Action    string    `gorm:"column:action;type:varchar(64);not null"`
	BundleId  string    `gorm:"column:bundle_id;type:varchar(64);index"`
	ChainTxID string    `gorm:"column:chain_tx_id;type:varchar(128)"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	ActionAt  time.Time `gorm:"column:action_at;not null"`
}

func (AuditEvent) TableName() string {
	return "access_audit_cache"
}
