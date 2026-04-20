package model

import (
	"time"

	"gorm.io/gorm"
)

// Server 项目下的主机资产：SSH 地址、来源（自建/云同步）、云元数据与连通性探测结果。
type Server struct {
	ID uint `json:"id" gorm:"primaryKey;index:idx_servers_project_id_id,priority:2;comment:主键ID"`

	ProjectID       uint   `json:"project_id" gorm:"not null;index;index:idx_servers_proj_group_source_deleted,priority:1;index:idx_servers_project_id_id,priority:1;comment:所属项目ID"`
	GroupID         *uint  `json:"group_id,omitempty" gorm:"index;index:idx_servers_proj_group_source_deleted,priority:2;comment:所属分组ID"`
	Name            string `json:"name" gorm:"size:128;not null;comment:服务器名称"`
	Host            string `json:"host" gorm:"size:255;not null;index;comment:服务器主机地址"`
	Port            int    `json:"port" gorm:"not null;default:22;comment:SSH端口"`
	OSType          string `json:"os_type" gorm:"size:32;not null;default:'linux';comment:操作系统类型"`
	OSArch          string `json:"os_arch" gorm:"size:32;not null;default:'';comment:操作系统架构"`
	Tags            string `json:"tags" gorm:"type:text;comment:服务器标签"` // comma separated
	Status          int    `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	SourceType      string `json:"source_type" gorm:"size:24;not null;default:'self_hosted';index;index:idx_servers_proj_group_source_deleted,priority:3;comment:来源类型"`
	Provider        string `json:"provider" gorm:"size:32;not null;default:'';index;comment:云厂商标识"`
	CloudInstanceID string `json:"cloud_instance_id" gorm:"size:128;not null;default:'';index;comment:云实例ID"`
	CloudRegion     string `json:"cloud_region" gorm:"size:64;not null;default:'';comment:云实例地域"`

	LastSeenAt    *time.Time `json:"last_seen_at" gorm:"comment:最近在线时间"`
	LastTestAt    *time.Time `json:"last_test_at" gorm:"comment:最近连通性测试时间"`
	LastTestError *string    `json:"last_test_error" gorm:"type:text;comment:最近测试错误信息"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_servers_proj_group_source_deleted,priority:4;comment:删除时间"`
}

// TableName 指定 GORM 表名为 servers。
func (Server) TableName() string { return "servers" }
