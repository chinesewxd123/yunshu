package model

import (
	"time"

	"gorm.io/gorm"
)

// Service 部署在某台服务器上的应用/服务实例，可带环境与标签。
type Service struct {
	ID uint `json:"id" gorm:"primaryKey;index:idx_services_server_id_id,priority:2;comment:主键ID"`

	ServerID uint    `json:"server_id" gorm:"not null;index;index:idx_services_server_id_id,priority:1;comment:所属服务器ID"`
	Name     string  `json:"name" gorm:"size:128;not null;index;comment:服务名称"`
	Env      *string `json:"env" gorm:"size:64;comment:部署环境"` // prod/stage/dev etc
	Labels   *string `json:"labels" gorm:"type:text;comment:服务标签"`
	Remark   *string `json:"remark" gorm:"type:text;comment:备注"`
	Status   int     `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 services。
func (Service) TableName() string { return "services" }
