package entity

import "time"

type AuditEvent struct {
	ID        uint      `gorm:"primaryKey"`
	Func      string    `gorm:"size:64;index"`
	ArgsJSON  string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
