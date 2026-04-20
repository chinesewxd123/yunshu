package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertRuleAssignee 监控规则处理人：关联用户与部门（部门内启用用户），用于通知扩展与排班展示。
type AlertRuleAssignee struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	MonitorRuleID       uint   `json:"monitor_rule_id" gorm:"not null;index;comment:监控规则ID"`
	UserIDsJSON         string `json:"user_ids_json" gorm:"type:text;comment:用户 ID 数组 JSON，如 [1,2]"`
	DepartmentIDsJSON   string `json:"department_ids_json" gorm:"type:text;comment:部门 ID 数组 JSON，如 [3]"`
	ExtraEmailsJSON     string `json:"extra_emails_json" gorm:"type:text;comment:额外邮箱 JSON 数组"`
	NotifyOnResolved    bool   `json:"notify_on_resolved" gorm:"not null;default:false;comment:是否在恢复时通知处理人"`
	Remark              string `json:"remark" gorm:"size:512;comment:备注"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertRuleAssignee) TableName() string {
	return "alert_rule_assignees"
}
