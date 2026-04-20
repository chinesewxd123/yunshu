package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户：登录名、邮箱、密码哈希、昵称、启用状态及关联角色（多对多）。
type User struct {
	ID           uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	Username     string         `json:"username" gorm:"size:64;not null;uniqueIndex;comment:用户名"`
	Email        *string        `json:"email" gorm:"size:128;index;comment:邮箱"`
	Password     string         `json:"-" gorm:"size:255;not null;comment:加密后的登录密码"`
	Nickname     string         `json:"nickname" gorm:"size:128;not null;comment:昵称"`
	Status       int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	DepartmentID *uint          `json:"department_id" gorm:"index;comment:所属部门ID"`
	Department   *Department    `json:"department,omitempty" gorm:"foreignKey:DepartmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Roles        []Role         `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt    time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}
