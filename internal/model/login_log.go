package model

import "time"

const (
	LoginLogStatusSuccess = 1
	LoginLogStatusFail    = 0
)

// 登录来源：用户名密码（含图形验证码）或邮箱验证码
const (
	LoginSourcePassword = "password"
	LoginSourceEmail    = "email"
)

// LoginLog 用户登录审计：成功/失败、来源、IP、UA 及可选关联用户。
type LoginLog struct {
	ID        uint      `json:"id" gorm:"primaryKey;index:idx_loginlogs_source_status_id,priority:3;comment:主键ID"`
	CreatedAt time.Time `json:"created_at" gorm:"index;index:idx_loginlogs_status_created_at,priority:2;comment:创建时间"`
	Username  string    `json:"username" gorm:"size:128;index;comment:登录用户名"`
	IP        string    `json:"ip" gorm:"size:64;comment:登录IP"`
	Status    int       `json:"status" gorm:"index;index:idx_loginlogs_source_status_id,priority:2;index:idx_loginlogs_status_created_at,priority:1;not null;comment:登录状态"` // 1 成功 0 失败
	Detail    string    `json:"detail" gorm:"size:512;comment:登录详情"`
	UserAgent string    `json:"user_agent" gorm:"type:text;comment:客户端标识"`
	Source    string    `json:"source" gorm:"size:32;index;index:idx_loginlogs_source_status_id,priority:1;not null;comment:登录来源"` // password | email
	UserID    *uint     `json:"user_id,omitempty" gorm:"comment:关联用户ID"`
}

// TableName 指定 GORM 表名为 login_logs。
func (LoginLog) TableName() string {
	return "login_logs"
}
