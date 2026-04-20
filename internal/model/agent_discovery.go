package model

import (
	"time"

	"gorm.io/gorm"
)

// AgentDiscovery Agent 在主机上发现的日志候选项（路径/单元等），用于配置向导。
type AgentDiscovery struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	ServerID  uint `json:"server_id" gorm:"not null;index;comment:所属服务器ID"`

	Kind  string `json:"kind" gorm:"size:16;not null;comment:发现类型"` // file/dir/unit
	Value string `json:"value" gorm:"type:text;not null;comment:发现值"`

	// Extra stores lightweight metadata (json string): e.g. size, mtime, hint.
	Extra *string `json:"extra" gorm:"type:text;comment:附加元数据"`

	FirstSeenAt time.Time `json:"first_seen_at" gorm:"not null;index;comment:首次发现时间"`
	LastSeenAt  time.Time `json:"last_seen_at" gorm:"not null;index;comment:最后发现时间"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 agent_discoveries。
func (AgentDiscovery) TableName() string { return "agent_discoveries" }
