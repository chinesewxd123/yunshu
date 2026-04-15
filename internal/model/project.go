package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID uint `json:"id" gorm:"primaryKey"`

	Name        string  `json:"name" gorm:"size:128;not null;index"`
	Code        string  `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Description *string `json:"description" gorm:"type:text"`
	Status      int     `json:"status" gorm:"not null;default:1"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Project) TableName() string { return "projects" }
