package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertMonitorRule 平台侧监控告警规则：周期性对 Prometheus 执行 PromQL，满足 for 后产生告警并走现有 Webhook/通道链路。
type AlertMonitorRule struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	DatasourceID uint   `json:"datasource_id" gorm:"not null;index;comment:告警数据源ID"`
	Name         string `json:"name" gorm:"size:128;not null;index;comment:规则名称"`
	Expr         string `json:"expr" gorm:"type:text;not null;comment:PromQL 表达式"`

	ForSeconds          int `json:"for_seconds" gorm:"not null;default:0;comment:持续满足秒数，类比 Prometheus for"`
	EvalIntervalSeconds int `json:"eval_interval_seconds" gorm:"not null;default:30;comment:评估间隔秒"`

	Severity        string `json:"severity" gorm:"size:32;not null;default:warning;comment:告警级别"`
	ThresholdUnit   string `json:"threshold_unit" gorm:"size:32;not null;default:raw;comment:阈值单位，如 percent/bytes/ms/count/raw"`
	LabelsJSON      string `json:"labels_json" gorm:"type:text;comment:附加 labels JSON"`
	AnnotationsJSON string `json:"annotations_json" gorm:"type:text;comment:附加 annotations JSON"`
	Enabled         bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertMonitorRule) TableName() string {
	return "alert_monitor_rules"
}
