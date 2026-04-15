package model

import (
	"time"

	"gorm.io/gorm"
)

type LogAgent struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ProjectID uint   `json:"project_id" gorm:"not null;index"`
	ServerID  uint   `json:"server_id" gorm:"not null;uniqueIndex"`
	Name      string `json:"name" gorm:"size:128;not null"`
	Version   string `json:"version" gorm:"size:64"`

	TokenHash string `json:"-" gorm:"size:64;not null;uniqueIndex"`
	Status    int    `json:"status" gorm:"not null;default:1"`

	LastSeenAt *time.Time `json:"last_seen_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (LogAgent) TableName() string { return "log_agents" }
