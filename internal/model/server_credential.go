package model

import (
	"time"

	"gorm.io/gorm"
)

// ServerCredential stores encrypted auth info for SSH connections.
// All secret fields must be encrypted before persisted.
type ServerCredential struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ServerID uint   `json:"server_id" gorm:"not null;uniqueIndex"`
	AuthType string `json:"auth_type" gorm:"size:16;not null;default:'password'"` // password/key
	Username string `json:"username" gorm:"size:128;not null"`

	EncPassword    *string `json:"-" gorm:"type:longtext"`
	EncPrivateKey  *string `json:"-" gorm:"type:longtext"`
	EncPassphrase  *string `json:"-" gorm:"type:longtext"`
	KeyVersion     int     `json:"key_version" gorm:"not null;default:1"`
	FingerprintSHA string  `json:"fingerprint_sha" gorm:"size:64;index"` // optional: for audit / rotation

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServerCredential) TableName() string { return "server_credentials" }
