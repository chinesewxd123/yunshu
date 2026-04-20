package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertPolicy 告警策略：控制匹配条件、通知通道和抑制窗口。
type AlertPolicy struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name        string `json:"name" gorm:"size:128;not null;uniqueIndex;comment:策略名称"`
	Description string `json:"description" gorm:"type:text;comment:策略描述"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Priority    int    `json:"priority" gorm:"not null;default:100;index;comment:优先级，越小越高"`

	// 匹配规则（JSON）
	MatchLabelsJSON string `json:"match_labels_json" gorm:"type:text;comment:精确匹配labels JSON"`
	MatchRegexJSON  string `json:"match_regex_json" gorm:"type:text;comment:正则匹配labels JSON"`
	ChannelsJSON    string `json:"channels_json" gorm:"type:text;comment:通知通道ID数组JSON"`

	TemplateID *uint `json:"template_id" gorm:"index;comment:关联模板ID(可选)"`

	NotifyResolved bool `json:"notify_resolved" gorm:"not null;default:true;comment:是否通知恢复"`
	SilenceSeconds int  `json:"silence_seconds" gorm:"not null;default:0;comment:同策略静默窗口秒"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertPolicy) TableName() string {
	return "alert_policies"
}
