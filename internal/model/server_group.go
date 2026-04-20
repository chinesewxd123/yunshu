package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	// ServerGroupCategorySelfHosted 自建服务器分组。
	ServerGroupCategorySelfHosted = "self_hosted"
	// ServerGroupCategoryCloud 云资源同步分组。
	ServerGroupCategoryCloud = "cloud"
)

// ServerGroup 服务器树形分组：支持父子层级，区分自建与云同步类别及云厂商。
type ServerGroup struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint  `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	ParentID  *uint `json:"parent_id,omitempty" gorm:"index;comment:父分组ID"`

	Name     string `json:"name" gorm:"size:128;not null;comment:分组名称"`
	Category string `json:"category" gorm:"size:24;not null;default:'self_hosted';index;comment:分组类型"` // self_hosted | cloud
	Provider string `json:"provider" gorm:"size:32;not null;default:'';index;comment:云厂商标识"`           // alibaba/tencent/jd/custom
	Sort     int    `json:"sort" gorm:"not null;default:0;comment:排序值"`
	Status   int    `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 server_groups。
func (ServerGroup) TableName() string { return "server_groups" }
