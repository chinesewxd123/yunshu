package model

import (
	"time"

	"gorm.io/gorm"
)

type Server struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ProjectID uint   `json:"project_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"size:128;not null"`
	Host      string `json:"host" gorm:"size:255;not null;index"`
	Port      int    `json:"port" gorm:"not null;default:22"`
	OSType    string `json:"os_type" gorm:"size:32;not null;default:'linux'"`
	OSArch    string `json:"os_arch" gorm:"size:32;not null;default:''"`
	Tags      string `json:"tags" gorm:"type:text"` // comma separated
	Status    int    `json:"status" gorm:"not null;default:1"`

	LastSeenAt    *time.Time `json:"last_seen_at"`
	LastTestAt    *time.Time `json:"last_test_at"`
	LastTestError *string    `json:"last_test_error" gorm:"type:text"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Server) TableName() string { return "servers" }
