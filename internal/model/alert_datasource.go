package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertDatasource 告警数据源（首期 Prometheus）：平台内可配置多条，供 PromQL 与监控规则使用。
type AlertDatasource struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index;comment:所属项目ID"`

	Name string `json:"name" gorm:"size:128;not null;index;comment:显示名称"`
	Type string `json:"type" gorm:"size:32;not null;default:prometheus;index;comment:类型 prometheus"`

	BaseURL       string `json:"base_url" gorm:"size:512;not null;comment:API 根地址，如 http://prom:9090"`
	BearerToken   string `json:"bearer_token,omitempty" gorm:"type:text;comment:Bearer Token"`
	BasicUser     string `json:"basic_user,omitempty" gorm:"size:128;comment:Basic 用户名"`
	BasicPassword string `json:"basic_password,omitempty" gorm:"size:256;comment:Basic 密码"`
	SkipTLSVerify bool   `json:"skip_tls_verify" gorm:"not null;default:false;comment:跳过 TLS 校验（仅内网调试）"`

	Enabled bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Remark  string `json:"remark" gorm:"size:512;comment:备注"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertDatasource) TableName() string {
	return "alert_datasources"
}
