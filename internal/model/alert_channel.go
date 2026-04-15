package model

import (
	"time"

	"gorm.io/gorm"
)

type AlertChannel struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex"`
	Type        string         `json:"type" gorm:"size:32;not null;default:generic_webhook"`
	URL         string         `json:"url" gorm:"size:1024;not null"`
	Secret      string         `json:"secret" gorm:"size:256"`
	HeadersJSON string         `json:"headers_json" gorm:"type:text"`
	Enabled     bool           `json:"enabled" gorm:"not null;default:true"`
	TimeoutMS   int            `json:"timeout_ms" gorm:"not null;default:5000"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertChannel) TableName() string {
	return "alert_channels"
}
