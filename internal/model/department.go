package model

import (
	"time"

	"gorm.io/gorm"
)

// Department 组织架构部门，采用物化路径（ancestors）表示层级，支持稳定的树形查询和子树迁移。
type Department struct {
	ID        uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	ParentID  *uint          `json:"parent_id" gorm:"index;index:idx_departments_parent_sort_deleted,priority:1;comment:父部门ID"`
	Name      string         `json:"name" gorm:"size:128;not null;comment:部门名称"`
	Code      string         `json:"code" gorm:"size:64;not null;uniqueIndex;comment:部门编码"`
	Ancestors string         `json:"ancestors" gorm:"size:512;not null;default:'/';index;comment:祖先路径，如 /1/3/"`
	Level     int            `json:"level" gorm:"not null;default:1;comment:层级，根节点为1"`
	Sort      int            `json:"sort" gorm:"not null;default:0;index:idx_departments_parent_sort_deleted,priority:2;comment:同级排序"`
	Status    int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	LeaderID  *uint          `json:"leader_id" gorm:"index;comment:部门负责人用户ID"`
	Phone     string         `json:"phone" gorm:"size:32;comment:联系电话"`
	Email     string         `json:"email" gorm:"size:128;comment:联系邮箱"`
	Remark    string         `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_departments_parent_sort_deleted,priority:3;comment:删除时间"`
	Children  []Department   `json:"children,omitempty" gorm:"-"`
}

// TableName 指定 GORM 表名为 departments。
func (Department) TableName() string {
	return "departments"
}
