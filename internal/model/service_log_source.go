package model

import (
	"time"

	"gorm.io/gorm"
)

// ServiceLogSource 服务的日志采集配置：文件路径或 journal 单元及过滤规则。
type ServiceLogSource struct {
	ID uint `json:"id" gorm:"primaryKey;index:idx_log_sources_service_id_id,priority:2;comment:主键ID"`

	ServiceID uint   `json:"service_id" gorm:"not null;index;index:idx_log_sources_service_id_id,priority:1;comment:所属服务ID"`
	LogType   string `json:"log_type" gorm:"size:16;not null;default:'file';comment:日志类型"` // file/journal
	Path      string `json:"path" gorm:"type:text;not null;comment:日志路径或日志单元"`             // file path or journal unit

	Encoding      *string `json:"encoding" gorm:"size:32;comment:日志编码"`
	Timezone      *string `json:"timezone" gorm:"size:64;comment:日志时区"`
	MultilineRule *string `json:"multiline_rule" gorm:"type:text;comment:多行合并规则"`
	IncludeRegex  *string `json:"include_regex" gorm:"type:text;comment:包含过滤正则"`
	ExcludeRegex  *string `json:"exclude_regex" gorm:"type:text;comment:排除过滤正则"`

	Status int `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 service_log_sources。
func (ServiceLogSource) TableName() string { return "service_log_sources" }
