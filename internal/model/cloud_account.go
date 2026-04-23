package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CloudAccount 云厂商 API 凭证（AK/SK 密文）与同步范围，用于拉取云主机。
type CloudAccount struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	GroupID   uint `json:"group_id" gorm:"not null;index;comment:所属云分组ID"`

	Provider    string `json:"provider" gorm:"size:32;not null;index;comment:云厂商标识"` // alibaba/tencent/jd/custom
	AccountName string `json:"account_name" gorm:"size:128;not null;comment:云账号名称"`
	RegionScope string `json:"region_scope" gorm:"size:256;not null;default:'';comment:同步地域范围"` // comma-separated regions

	EncAK       *string `json:"-" gorm:"type:longtext;comment:加密后的访问密钥AK"`
	EncSK       *string `json:"-" gorm:"type:longtext;comment:加密后的访问密钥SK"`
	AKDictLabel *string `json:"ak_dict_label,omitempty" gorm:"size:191;comment:AK 字典标签（用于回显）"`
	SKDictLabel *string `json:"sk_dict_label,omitempty" gorm:"size:191;comment:SK 字典标签（用于回显）"`

	ExtraConfig datatypes.JSON `json:"extra_config" gorm:"type:json;comment:扩展配置"`
	Status      int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	LastSyncAt    *time.Time `json:"last_sync_at,omitempty" gorm:"comment:最近同步时间"`
	LastSyncError *string    `json:"last_sync_error,omitempty" gorm:"type:text;comment:最近同步错误"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 cloud_accounts。
func (CloudAccount) TableName() string { return "cloud_accounts" }
