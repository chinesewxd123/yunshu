package model

import (
	"time"

	"gorm.io/gorm"
)

type Permission struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64;not null"`
	Resource    string         `json:"resource" gorm:"size:191;not null;uniqueIndex:idx_resource_action"`
	Action      string         `json:"action" gorm:"size:32;not null;uniqueIndex:idx_resource_action"`
	Description string         `json:"description" gorm:"size:255"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
