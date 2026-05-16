package model

import (
	"time"

	"gorm.io/gorm"
)

// UserGroup 用户组：可对组授予集群档位，组内成员继承（对齐 k8m 多集群权限中的用户组）。
// ScopeProjectID 非空时表示该组仅在该项目上下文中使用（如通知/K8s 授权时与项目绑定）；为空表示全局组。
type UserGroup struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64;not null;comment:显示名称"`
	Code        string         `json:"code" gorm:"size:64;not null;uniqueIndex;comment:唯一编码"`
	Description string         `json:"description" gorm:"size:255;comment:说明"`
	ScopeProjectID *uint       `json:"scope_project_id,omitempty" gorm:"index;comment:可选：绑定业务项目ID，非空时仅项目成员可维护该组"`
	Status      int            `json:"status" gorm:"not null;default:1;comment:1启用 0停用"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserGroup) TableName() string { return "user_groups" }
