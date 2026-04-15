package model

import (
	"time"

	"gorm.io/gorm"
)

type AlertEvent struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Source          string         `json:"source" gorm:"size:64;not null;index"`
	Title           string         `json:"title" gorm:"size:255;not null"`
	Severity        string         `json:"severity" gorm:"size:32;not null;default:warning"`
	Status          string         `json:"status" gorm:"size:32;not null;default:firing"`
	Cluster         string         `json:"cluster" gorm:"size:128;index"`
	GroupKey        string         `json:"group_key" gorm:"size:128;index"`
	LabelsDigest    string         `json:"labels_digest" gorm:"size:128;index"`
	ChannelID       uint           `json:"channel_id" gorm:"index"`
	ChannelName     string         `json:"channel_name" gorm:"size:64"`
	Success         bool           `json:"success" gorm:"not null;default:false"`
	HTTPStatusCode  int            `json:"http_status_code"`
	ErrorMessage    string         `json:"error_message" gorm:"size:1024"`
	RequestPayload  string         `json:"request_payload" gorm:"type:longtext"`
	ResponsePayload string         `json:"response_payload" gorm:"type:longtext"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertEvent) TableName() string {
	return "alert_events"
}
