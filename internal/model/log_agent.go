package model

import (
	"time"

	"gorm.io/gorm"
)

// LogAgent 项目内日志采集 Agent：与服务器一对一，令牌哈希鉴权与心跳。
type LogAgent struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint   `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	ServerID  uint   `json:"server_id" gorm:"not null;uniqueIndex;comment:所属服务器ID"`
	Name      string `json:"name" gorm:"size:128;not null;comment:Agent名称"`
	Version   string `json:"version" gorm:"size:64;comment:Agent版本"`

	TokenHash string `json:"-" gorm:"size:64;not null;uniqueIndex;comment:认证令牌哈希"`
	Status    int    `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`

	LastSeenAt      *time.Time `json:"last_seen_at" gorm:"comment:最近心跳时间"`
	ListenPort      int        `json:"listen_port" gorm:"not null;default:0;comment:本机对外监听端口，0 表示当前实现不监听（仅出站连接平台 gRPC）"`
	InstallProgress int        `json:"install_progress" gorm:"not null;default:0;comment:安装进度0-100"`
	HealthStatus    string     `json:"health_status" gorm:"size:32;default:unknown;comment:健康状态"`
	LastError       string     `json:"last_error" gorm:"size:1024;comment:最近错误"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 log_agents。
func (LogAgent) TableName() string { return "log_agents" }
