package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertSilence 告警静默： matchers 为 JSON 数组，语义接近 Alertmanager matcher（全部命中即静默）。
type AlertSilence struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name         string    `json:"name" gorm:"size:128;not null;comment:静默名称"`
	MatchersJSON string    `json:"matchers_json" gorm:"type:text;not null;comment:匹配器 JSON，如 [{\"name\":\"alertname\",\"value\":\"x\",\"is_regex\":false}]"`
	StartsAt     time.Time `json:"starts_at" gorm:"index;comment:生效开始时间"`
	EndsAt       time.Time `json:"ends_at" gorm:"index;comment:生效结束时间"`
	Comment      string    `json:"comment" gorm:"size:512;comment:说明"`
	CreatedBy    uint      `json:"created_by" gorm:"index;comment:创建人用户ID"`
	Enabled      bool      `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertSilence) TableName() string {
	return "alert_silences"
}
