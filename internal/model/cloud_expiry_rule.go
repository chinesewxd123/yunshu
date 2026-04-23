package model

import (
	"time"

	"gorm.io/gorm"
)

// CloudExpiryRule 云资源到期规则：周期拉取云实例到期时间并触发告警。
type CloudExpiryRule struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID   uint   `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	Name        string `json:"name" gorm:"size:128;not null;index;comment:规则名称"`
	Provider    string `json:"provider" gorm:"size:32;not null;default:'';index;comment:云厂商标识，空表示全部"`
	RegionScope string `json:"region_scope" gorm:"size:256;not null;default:'';comment:地域范围，逗号分隔，空表示全部"`

	AdvanceDays int    `json:"advance_days" gorm:"not null;default:7;comment:提前多少天告警"`
	Severity    string `json:"severity" gorm:"size:32;not null;default:warning;comment:告警级别"`
	LabelsJSON  string `json:"labels_json" gorm:"type:text;comment:附加 labels JSON"`

	EvalIntervalSeconds int  `json:"eval_interval_seconds" gorm:"not null;default:3600;comment:评估间隔秒"`
	Enabled             bool `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CloudExpiryRule) TableName() string { return "cloud_expiry_rules" }
