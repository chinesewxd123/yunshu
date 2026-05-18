package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// K8sForwardedEvent 持久化的待转发 K8s 事件（参考 k8m k8s_events；Normal 不转发）。
type K8sForwardedEvent struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	EvtKey    string    `json:"evt_key" gorm:"size:255;uniqueIndex:idx_k8s_fwd_evt_key"`
	ClusterID string    `json:"cluster_id" gorm:"size:32;index:idx_k8s_fwd_cluster"`
	Namespace string    `json:"namespace" gorm:"size:64;index:idx_k8s_fwd_namespace"`
	Name      string    `json:"name" gorm:"size:255"`
	Type      string    `json:"type" gorm:"size:16"`
	Reason    string    `json:"reason" gorm:"size:128"`
	Level     string    `json:"level" gorm:"size:16"`
	Message   string    `json:"message" gorm:"type:text"`
	Timestamp time.Time `json:"timestamp" gorm:"index:idx_k8s_fwd_timestamp"`
	Processed bool      `json:"processed" gorm:"default:false;index:idx_k8s_fwd_processed"`
	Attempts  int       `json:"-" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"<-:create"`
	UpdatedAt time.Time `json:"-"`
}

func (K8sForwardedEvent) TableName() string { return "k8s_forwarded_events" }

// ShouldForward 是否应转发：K8s Event 标准类型为 Normal / Warning，Normal 不发送，其余均发送。
func (e *K8sForwardedEvent) ShouldForward() bool {
	if e == nil {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(e.Type), "Normal")
}

// K8sEventForwardRule 多集群事件转发规则（参考 k8m k8s_event_config）。
type K8sEventForwardRule struct {
	ID          uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"type:text"`
	// ClusterIDs 逗号分隔的集群 ID，如 "1,2,3"
	ClusterIDs string `json:"cluster_ids" gorm:"type:text"`
	// WebhookURL 目标地址；留空/internal/alertmanager 时 POST 本机 /alerts/webhook/alertmanager（与告警平台共用）
	WebhookURL string `json:"webhook_url" gorm:"size:512"`
	Enabled    bool   `json:"enabled" gorm:"default:true;index"`

	RuleNamespaces string `json:"rule_namespaces" gorm:"type:text"` // JSON []string 精确匹配 namespace
	RuleNames      string `json:"rule_names" gorm:"type:text"`      // JSON []string 子串匹配 involved object name
	RuleReasons    string `json:"rule_reasons" gorm:"type:text"`    // JSON []string 子串匹配 reason/message
	RuleReverse    bool   `json:"rule_reverse" gorm:"default:false"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (K8sEventForwardRule) TableName() string { return "k8s_event_forward_rules" }

// K8sEventForwardSetting 全局转发参数（单行，id=1）。
type K8sEventForwardSetting struct {
	ID                     uint `json:"id" gorm:"primaryKey"`
	ProcessIntervalSeconds int  `json:"process_interval_seconds" gorm:"default:10"`
	BatchSize              int  `json:"batch_size" gorm:"default:50"`
	MaxRetries             int  `json:"max_retries" gorm:"default:3"`
	WatcherBufferSize      int  `json:"watcher_buffer_size" gorm:"default:1000"`
}

func (K8sEventForwardSetting) TableName() string { return "k8s_event_forward_settings" }
