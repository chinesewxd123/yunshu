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

type LoginLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username" gorm:"size:128;index"`
	IP        string    `json:"ip" gorm:"size:64"`
	Status    int       `json:"status" gorm:"index;not null"` // 1 成功 0 失败
	Detail    string    `json:"detail" gorm:"size:512"`
	UserAgent string    `json:"user_agent" gorm:"type:text"`
	Source    string    `json:"source" gorm:"size:32;index;not null"` // password | email
	UserID    *uint     `json:"user_id,omitempty"`
}

func (LoginLog) TableName() string {
	return "login_logs"
}
