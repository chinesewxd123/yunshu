package model

import (
	"time"

	"gorm.io/gorm"
)

// ServiceLogSource describes where to read logs for a service.
type ServiceLogSource struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ServiceID uint   `json:"service_id" gorm:"not null;index"`
	LogType   string `json:"log_type" gorm:"size:16;not null;default:'file'"` // file/journal
	Path      string `json:"path" gorm:"type:text;not null"`                  // file path or journal unit

	Encoding      *string `json:"encoding" gorm:"size:32"`
	Timezone      *string `json:"timezone" gorm:"size:64"`
	MultilineRule *string `json:"multiline_rule" gorm:"type:text"`
	IncludeRegex  *string `json:"include_regex" gorm:"type:text"`
	ExcludeRegex  *string `json:"exclude_regex" gorm:"type:text"`

	Status int `json:"status" gorm:"not null;default:1"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ServiceLogSource) TableName() string { return "service_log_sources" }

