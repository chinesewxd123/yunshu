package model

import "time"

type UserRole struct {
	UserID    uint      `gorm:"primaryKey"`
	RoleID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}
