package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertChannel 告警外发渠道：Webhook URL、密钥、自定义头与超时等。
type AlertChannel struct {
	ID          uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex;comment:渠道名称"`
	Type        string         `json:"type" gorm:"size:32;not null;default:generic_webhook;comment:渠道类型"`
	URL         string         `json:"url" gorm:"size:1024;not null;comment:回调地址"`
	Secret      string         `json:"secret" gorm:"size:256;comment:签名密钥"`
	HeadersJSON string         `json:"headers_json" gorm:"type:text;comment:自定义请求头JSON"`
	Enabled     bool           `json:"enabled" gorm:"not null;default:true;comment:是否启用"`
	TimeoutMS   int            `json:"timeout_ms" gorm:"not null;default:5000;comment:超时时间毫秒"`
	CreatedAt   time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 alert_channels。
func (AlertChannel) TableName() string {
	return "alert_channels"
}
