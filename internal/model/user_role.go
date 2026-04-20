package model

import "time"

// UserRole 用户与角色的关联表（复合主键）。
type UserRole struct {
	UserID    uint      `gorm:"primaryKey;comment:用户ID"`
	RoleID    uint      `gorm:"primaryKey;comment:角色ID"`
	CreatedAt time.Time `json:"created_at" gorm:"comment:创建时间"`
}
