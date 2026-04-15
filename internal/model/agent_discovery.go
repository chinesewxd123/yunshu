package model

import (
	"time"

	"gorm.io/gorm"
)

// AgentDiscovery stores log candidates discovered by log-agent on a specific server.
// It is used to drive "agent-first" configuration UX (pick from discovered paths/units).
type AgentDiscovery struct {
	ID uint `json:"id" gorm:"primaryKey"`

	ProjectID uint `json:"project_id" gorm:"not null;index"`
	ServerID  uint `json:"server_id" gorm:"not null;index"`

	Kind  string `json:"kind" gorm:"size:16;not null"` // file/dir/unit
	Value string `json:"value" gorm:"type:text;not null"`

	// Extra stores lightweight metadata (json string): e.g. size, mtime, hint.
	Extra *string `json:"extra" gorm:"type:text"`

	FirstSeenAt time.Time `json:"first_seen_at" gorm:"not null;index"`
	LastSeenAt  time.Time `json:"last_seen_at" gorm:"not null;index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentDiscovery) TableName() string { return "agent_discoveries" }

