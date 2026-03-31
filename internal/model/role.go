package model

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex"`
	Code        string         `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Description string         `json:"description" gorm:"size:255"`
	Status      int            `json:"status" gorm:"not null;default:1"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func ExtractRoleCodes(roles []Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}
