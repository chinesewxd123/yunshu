package model

import "time"

// RegistrationRequestStatus 用户自助注册申请的审核状态（待审/通过/拒绝）。
type RegistrationRequestStatus int

const (
	RegistrationPending  RegistrationRequestStatus = 0
	RegistrationApproved RegistrationRequestStatus = 1
	RegistrationRejected RegistrationRequestStatus = 2
)

// RegistrationRequest 自助注册申请单：账号信息与审核人、审核意见、时间戳。
type RegistrationRequest struct {
	ID            uint                      `json:"id" gorm:"primaryKey;comment:主键ID"`
	Username      string                    `json:"username" gorm:"size:64;not null;uniqueIndex;comment:申请用户名"`
	Email         string                    `json:"email" gorm:"size:128;not null;uniqueIndex;comment:申请邮箱"`
	Nickname      string                    `json:"nickname" gorm:"size:128;not null;comment:申请昵称"`
	Password      string                    `json:"-" gorm:"size:255;not null;comment:加密后的登录密码"`
	Status        RegistrationRequestStatus `json:"status" gorm:"not null;default:0;index;comment:审核状态 0待审核 1通过 2拒绝"`
	ReviewerID    *uint                     `json:"reviewer_id,omitempty" gorm:"comment:审核人用户ID"`
	ReviewComment string                    `json:"review_comment,omitempty" gorm:"size:255;comment:审核备注"`
	ReviewedAt    *time.Time                `json:"reviewed_at,omitempty" gorm:"comment:审核时间"`
	CreatedAt     time.Time                 `json:"created_at" gorm:"index;comment:创建时间"`
	UpdatedAt     time.Time                 `json:"updated_at" gorm:"comment:更新时间"`
}

// TableName 指定 GORM 表名为 registration_requests。
func (RegistrationRequest) TableName() string {
	return "registration_requests"
}
