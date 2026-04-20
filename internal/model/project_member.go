package model

import (
	"time"

	"gorm.io/gorm"
)

// ProjectMember 项目成员：用户与项目的多对多关系及项目内角色（与全局 RBAC 独立，用于资源归属与告警通知范围）。
// 角色语义建议：owner 负责人、admin 可管资源、member 默认参与、readonly 只读。
type ProjectMember struct {
	ID        uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	ProjectID uint   `json:"project_id" gorm:"not null;uniqueIndex:uniq_project_user;index;comment:项目ID"`
	UserID    uint   `json:"user_id" gorm:"not null;uniqueIndex:uniq_project_user;index;comment:用户ID"`
	Role      string `json:"role" gorm:"size:32;not null;default:member;comment:项目内角色 owner/admin/member/readonly"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ProjectMember) TableName() string { return "project_members" }
