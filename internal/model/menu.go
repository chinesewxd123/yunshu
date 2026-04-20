package model

import (
	"time"

	"gorm.io/gorm"
)

// Menu 前端侧边栏/路由菜单树节点，支持父子层级、排序、隐藏及仅管理员可见。
type Menu struct {
	ID        uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	ParentID  *uint          `json:"parent_id" gorm:"index;index:idx_menus_parent_sort_deleted,priority:1;comment:父菜单ID"`
	Path      string         `json:"path" gorm:"size:128;comment:路由路径"`
	Name      string         `json:"name" gorm:"size:64;not null;comment:菜单名称"`
	Icon      string         `json:"icon" gorm:"size:64;comment:菜单图标"`
	AdminOnly bool           `json:"admin_only" gorm:"default:false;comment:是否仅管理员可见"`
	Sort      int            `json:"sort" gorm:"default:0;index:idx_menus_parent_sort_deleted,priority:2;comment:排序值"`
	Hidden    bool           `json:"hidden" gorm:"default:false;comment:是否隐藏"`
	Component string         `json:"component" gorm:"size:128;comment:前端组件路径"`
	Redirect  string         `json:"redirect" gorm:"size:128;comment:重定向路径"`
	Status    int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_menus_parent_sort_deleted,priority:3;comment:删除时间"`
	Children  []Menu         `json:"children,omitempty" gorm:"-"`
}

// TableName 指定 GORM 表名为 menus。
func (Menu) TableName() string {
	return "menus"
}
