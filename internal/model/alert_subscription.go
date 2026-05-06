package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertSubscriptionNode 订阅树节点：实现树形告警路由
// 参考：夜莺业务组订阅机制、Alertmanager 路由树
type AlertSubscriptionNode struct {
	ID        uint `json:"id" gorm:"primaryKey;comment:主键ID"`
	ProjectID uint `json:"project_id" gorm:"not null;index;comment:业务组ID"`

	// 树形结构
	ParentID *uint `json:"parent_id" gorm:"index;comment:父节点ID"`
	Level    int   `json:"level" gorm:"not null;default:0;comment:层级深度"`
	Path     string `json:"path" gorm:"size:256;index;comment:节点路径，如/1/2/3"`

	// 节点名称与标识
	Name string `json:"name" gorm:"size:128;not null;comment:节点名称"`
	Code string `json:"code" gorm:"size:64;index;comment:节点编码，用于API"`

	// 匹配条件（与父节点条件AND组合）
	MatchLabelsJSON string `json:"match_labels_json" gorm:"type:text;comment:精确匹配labels JSON"`
	MatchRegexJSON  string `json:"match_regex_json" gorm:"type:text;comment:正则匹配labels JSON"`
	MatchSeverity   string `json:"match_severity" gorm:"size:32;comment:匹配的严重级别"`

	// 路由行为
	Continue bool `json:"continue" gorm:"not null;default:false;comment:匹配成功后是否继续匹配子节点"`
	Enabled  bool `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	// 通知配置
	ReceiverGroupIDsJSON string `json:"receiver_group_ids_json" gorm:"type:text;comment:通知组ID数组JSON"`
	SilenceSeconds       int    `json:"silence_seconds" gorm:"not null;default:0;comment:静默窗口秒"`
	NotifyResolved       bool   `json:"notify_resolved" gorm:"not null;default:true;comment:是否通知恢复"`

	// 非数据库字段
	MatchLabels     map[string]string `json:"match_labels,omitempty" gorm:"-"`
	MatchRegex      map[string]string `json:"match_regex,omitempty" gorm:"-"`
	ReceiverGroupIDs []uint           `json:"receiver_group_ids,omitempty" gorm:"-"`
	Children        []AlertSubscriptionNode `json:"children,omitempty" gorm:"-"`
	Parent          *AlertSubscriptionNode  `json:"parent,omitempty" gorm:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertSubscriptionNode) TableName() string {
	return "alert_subscription_nodes"
}

// AlertReceiverGroup 通知接收组：绑定多个通道
type AlertReceiverGroup struct {
	ID          uint   `json:"id" gorm:"primaryKey;comment:主键ID"`
	ProjectID   uint   `json:"project_id" gorm:"not null;index;comment:业务组ID"`
	Name        string `json:"name" gorm:"size:128;not null;comment:组名称"`
	Description string `json:"description" gorm:"type:text;comment:描述"`

	// 通知方式配置
	ChannelIDsJSON string `json:"channel_ids_json" gorm:"type:text;comment:通知通道ID数组JSON"`
	EmailRecipientsJSON string `json:"email_recipients_json" gorm:"type:text;comment:额外邮箱地址JSON"`

	// 时间限制（可选）
	ActiveTimeStart *string `json:"active_time_start" gorm:"size:8;comment:生效开始时间 HH:mm"`
	ActiveTimeEnd   *string `json:"active_time_end" gorm:"size:8;comment:生效结束时间 HH:mm"`
	WeekdaysJSON    string  `json:"weekdays_json" gorm:"type:text;comment:生效星期JSON，如[1,2,3,4,5]"`

	// 升级相关
	EscalationLevel int  `json:"escalation_level" gorm:"not null;default:0;comment:升级层级，0=初始，1=一级升级"`
	Enabled         bool `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	// 非数据库字段
	ChannelIDs   []uint   `json:"channel_ids,omitempty" gorm:"-"`
	EmailRecipients []string `json:"email_recipients,omitempty" gorm:"-"`
	Weekdays     []int    `json:"weekdays,omitempty" gorm:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertReceiverGroup) TableName() string {
	return "alert_receiver_groups"
}

// AlertSubscriptionMatch 订阅匹配记录：用于审计
type AlertSubscriptionMatch struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	AlertFingerprint  string `json:"alert_fingerprint" gorm:"size:64;index;comment:告警指纹"`
	AlertName         string `json:"alert_name" gorm:"size:128;comment:告警名称"`
	SubscriptionPath  string `json:"subscription_path" gorm:"size:256;comment:匹配的路径"`
	MatchedNodeIDs    string `json:"matched_node_ids" gorm:"size:256;comment:匹配的节点ID列表"`
	ReceiverGroupIDs  string `json:"receiver_group_ids" gorm:"size:256;comment:最终通知组ID列表"`

	LabelsJSON    string `json:"labels_json" gorm:"type:text;comment:告警labels"`
	NotifiedCount int    `json:"notified_count" gorm:"comment:通知次数"`

	CreatedAt time.Time `json:"created_at"`
}

func (AlertSubscriptionMatch) TableName() string {
	return "alert_subscription_matches"
}
