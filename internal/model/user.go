package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"size:64;not null;uniqueIndex"`
	Password  string         `json:"-" gorm:"size:255;not null"`
	Nickname  string         `json:"nickname" gorm:"size:128;not null"`
	Status    int            `json:"status" gorm:"not null;default:1"`
	Roles     []Role         `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
