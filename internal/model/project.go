package model

import (
	"time"

	"gorm.io/gorm"
)

// Project 业务项目：名称、唯一编码、描述，用于资源与成员的租户隔离。
type Project struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name        string  `json:"name" gorm:"size:128;not null;index;comment:项目名称"`
	Code        string  `json:"code" gorm:"size:64;not null;uniqueIndex;comment:项目编码"`
	Description *string `json:"description" gorm:"type:text;comment:项目描述"`
	Status      int     `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	// OwnerDepartmentID 可选归属部门，用于组织维度筛选与报表（不自动决定成员权限）。
	OwnerDepartmentID *uint       `json:"owner_department_id,omitempty" gorm:"index;comment:可选归属部门ID"`
	OwnerDepartment   *Department `json:"owner_department,omitempty" gorm:"foreignKey:OwnerDepartmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 projects。
func (Project) TableName() string { return "projects" }
