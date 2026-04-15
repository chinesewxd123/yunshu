package model

import (
	"time"

	"gorm.io/gorm"
)

// Service represents an app/service running on a server.
type Service struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ServerID uint    `json:"server_id" gorm:"not null;index"`
	Name     string  `json:"name" gorm:"size:128;not null;index"`
	Env      *string `json:"env" gorm:"size:64"` // prod/stage/dev etc
	Labels   *string `json:"labels" gorm:"type:text"`
	Remark   *string `json:"remark" gorm:"type:text"`
	Status   int     `json:"status" gorm:"not null;default:1"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Service) TableName() string { return "services" }
