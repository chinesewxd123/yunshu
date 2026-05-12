package model

import (
	"time"

	"gorm.io/gorm"
)

// UserGroup 用户组：可对组授予集群档位，组内成员继承（对齐 k8m 多集群权限中的用户组）。
type UserGroup struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64;not null;comment:显示名称"`
	Code        string         `json:"code" gorm:"size:64;not null;uniqueIndex;comment:唯一编码"`
	Description string         `json:"description" gorm:"size:255;comment:说明"`
	Status      int            `json:"status" gorm:"not null;default:1;comment:1启用 0停用"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserGroup) TableName() string { return "user_groups" }
