package entity

import (
	"consentcc/internal/model"
	"time"
)

type User struct {
	ID        string     `json:"id,omitempty" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string     `json:"name,omitempty" gorm:"type:varchar;not null"`
	Role      model.Role `json:"role,omitempty" gorm:"type:varchar(20);not null"`
	CreatedAt time.Time  `json:"created_at,omitempty" gorm:"autoCreateTime"`
}

type Grantee struct {
	ID             string     `json:"id,omitempty" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name           string     `json:"name,omitempty" gorm:"type:varchar;not null"`
	Role           model.Role `json:"role,omitempty" gorm:"type:varchar(20);not null"`
	NationalIDHash string     `json:"national_id_hash,omitempty" gorm:"type:varchar(64);not null"`
}

type CitizenIdentity struct {
	ID             string    `json:"id,omitempty" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         string    `json:"user_id,omitempty" gorm:"type:uuid;index;not null"`
	CountryCode    string    `json:"country_code,omitempty" gorm:"type:varchar(4);not null"`
	NationalIDHash string    `json:"national_id_hash,omitempty" gorm:"type:varchar(64);not null"`
	CreatedAt      time.Time `json:"created_at,omitempty" gorm:"autoCreateTime"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type ViewUserQuery struct {
	UserId         string `json:"user_id,omitempty" gorm:"column:user_id;type:uuid"`
	Name           string `json:"name,omitempty" gorm:"column:name;type:varchar"`
	Role           string `json:"role,omitempty" gorm:"column:role;type:varchar(20)"`
	NationalIDHash string `json:"national_id_hash,omitempty" gorm:"column:national_id_hash;type:varchar(64);not null"`
}

type ReqRegisterUser struct {
	Role        model.Role `json:"role"`
	CountryCode string     `json:"country_code"`
	NationalID  string     `json:"national_id"`
	Name        string     `json:"name"`
}

type ReqLoginUser struct {
	NationalIDHash string `json:"national_id_hash,omitempty"`
	Password       string `json:"password,omitempty"`
}

func (User) TableName() string {
	return "users"
}

func (CitizenIdentity) TableName() string {
	return "citizen_identities"
}
