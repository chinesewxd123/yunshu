package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色定义：名称、唯一编码、描述及启用状态，用于 RBAC 与菜单权限过滤。
type Role struct {
	ID          uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex;comment:角色名称"`
	Code        string         `json:"code" gorm:"size:64;not null;uniqueIndex;comment:角色编码"`
	Description string         `json:"description" gorm:"size:255;comment:角色描述"`
	Status      int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	CreatedAt   time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// ExtractRoleCodes 从角色列表提取角色编码，用于鉴权与策略匹配。
func ExtractRoleCodes(roles []Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}
