package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertInhibitionRule 告警抑制规则：当源告警触发时，抑制目标告警的发送
// 参考：Alertmanager inhibit_rules / 夜莺告警屏蔽策略
type AlertInhibitionRule struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name        string `json:"name" gorm:"size:128;not null;index;comment:规则名称"`
	Description string `json:"description" gorm:"type:text;comment:规则描述"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Priority    int    `json:"priority" gorm:"not null;default:100;comment:优先级，越小越高"`

	// 源告警匹配条件（当这些告警触发时，产生抑制作用）
	SourceMatchLabelsJSON string `json:"source_match_labels_json" gorm:"type:text;comment:源告警精确匹配labels JSON"`
	SourceMatchRegexJSON  string `json:"source_match_regex_json" gorm:"type:text;comment:源告警正则匹配labels JSON"`

	// 目标告警匹配条件（被抑制的告警）
	TargetMatchLabelsJSON string `json:"target_match_labels_json" gorm:"type:text;comment:目标告警精确匹配labels JSON"`
	TargetMatchRegexJSON  string `json:"target_match_regex_json" gorm:"type:text;comment:目标告警正则匹配labels JSON"`

	// 必须相等的标签（源和目标在这些标签上值必须相同，才产生抑制）
	EqualLabelsJSON string `json:"equal_labels_json" gorm:"type:text;comment:必须相等的标签名数组JSON"`

	// 抑制持续时间
	DurationSeconds int `json:"duration_seconds" gorm:"not null;default:3600;comment:抑制持续时间秒"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 非数据库字段，缓存解析后的数据
	SourceMatchLabels map[string]string `json:"source_match_labels,omitempty" gorm:"-"`
	SourceMatchRegex  map[string]string `json:"source_match_regex,omitempty" gorm:"-"`
	TargetMatchLabels map[string]string `json:"target_match_labels,omitempty" gorm:"-"`
	TargetMatchRegex  map[string]string `json:"target_match_regex,omitempty" gorm:"-"`
	EqualLabels       []string          `json:"equal_labels,omitempty" gorm:"-"`
}

func (AlertInhibitionRule) TableName() string {
	return "alert_inhibition_rules"
}

// AlertInhibitionEvent 告警抑制事件记录：记录哪些告警被抑制
type AlertInhibitionEvent struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	RuleID        uint   `json:"rule_id" gorm:"index;comment:触发抑制的规则ID"`
	RuleName      string `json:"rule_name" gorm:"size:128;comment:规则名称"`

	SourceFingerprint string `json:"source_fingerprint" gorm:"size:64;index;comment:源告警指纹"`
	TargetFingerprint string `json:"target_fingerprint" gorm:"size:64;index;comment:被抑制告警指纹"`

	SourceAlertName string `json:"source_alert_name" gorm:"size:128;comment:源告警名称"`
	TargetAlertName string `json:"target_alert_name" gorm:"size:128;comment:被抑制告警名称"`

	EqualLabelValuesJSON string `json:"equal_label_values_json" gorm:"type:text;comment:相等标签值JSON"`

	StartedAt time.Time `json:"started_at" gorm:"comment:抑制开始时间"`
	EndedAt   time.Time `json:"ended_at" gorm:"comment:抑制结束时间"`

	CreatedAt time.Time `json:"created_at"`
}

func (AlertInhibitionEvent) TableName() string {
	return "alert_inhibition_events"
}

// AlertInhibitionActive 当前活跃的抑制状态（Redis存储结构参考）
type AlertInhibitionActive struct {
	SourceFingerprint string            `json:"source_fingerprint"`
	TargetFingerprint string            `json:"target_fingerprint"`
	RuleID            uint              `json:"rule_id"`
	EqualValues       map[string]string `json:"equal_values"`
	StartedAt         time.Time         `json:"started_at"`
	ExpiresAt         time.Time         `json:"expires_at"`
}
