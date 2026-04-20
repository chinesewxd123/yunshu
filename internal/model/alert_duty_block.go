package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertDutyBlock 值班班次块：[starts_at, ends_at] 内生效；可与监控规则关联的值班表组合使用。
type AlertDutyBlock struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	MonitorRuleID     uint      `json:"monitor_rule_id" gorm:"not null;index;comment:监控规则ID（直接绑定值班）"`
	StartsAt          time.Time `json:"starts_at" gorm:"index;comment:开始时间"`
	EndsAt            time.Time `json:"ends_at" gorm:"index;comment:结束时间"`
	Title             string    `json:"title" gorm:"size:128;comment:班次标题"`
	UserIDsJSON      string    `json:"user_ids_json" gorm:"type:text;comment:用户 ID JSON 数组"`
	DepartmentIDsJSON string    `json:"department_ids_json" gorm:"type:text;comment:部门根 ID JSON 数组（子树全员）"`
	ExtraEmailsJSON   string    `json:"extra_emails_json" gorm:"type:text;comment:额外邮箱 JSON"`
	Remark            string    `json:"remark" gorm:"size:512;comment:备注"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertDutyBlock) TableName() string {
	return "alert_duty_blocks"
}
