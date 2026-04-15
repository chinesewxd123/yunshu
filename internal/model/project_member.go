package model

import (
	"time"

	"gorm.io/gorm"
)

// ProjectMember binds a user to a project with a role scope.
// role: owner/admin/readonly
type ProjectMember struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ProjectID uint   `json:"project_id" gorm:"not null;index;uniqueIndex:uniq_project_user"`
	UserID    uint   `json:"user_id" gorm:"not null;index;uniqueIndex:uniq_project_user"`
	Role      string `json:"role" gorm:"size:32;not null;default:'readonly'"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ProjectMember) TableName() string { return "project_members" }
