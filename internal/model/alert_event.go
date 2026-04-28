package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertEvent 告警事件与通知投递记录：含渠道、集群、分组键及请求/响应快照。
type AlertEvent struct {
	ID                 uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	Source             string         `json:"source" gorm:"size:64;not null;index;comment:告警来源"`
	Title              string         `json:"title" gorm:"size:255;not null;comment:告警标题"`
	Severity           string         `json:"severity" gorm:"size:32;not null;default:warning;comment:严重级别"`
	Status             string         `json:"status" gorm:"size:32;not null;default:firing;comment:告警状态"`
	Cluster            string         `json:"cluster" gorm:"size:128;index;comment:K8s/Prometheus external_labels.cluster 等环境名；平台规则未显式配置时可为空"`
	Environment        string         `json:"environment,omitempty" gorm:"-"`
	AlertIP            string         `json:"alert_ip,omitempty" gorm:"-"`
	MonitorPipeline    string         `json:"monitor_pipeline" gorm:"size:32;index;comment:监控链路 prometheus=Prometheus+YAML+Alertmanager platform=平台规则"`
	GroupKey           string         `json:"group_key" gorm:"size:128;index;comment:聚合分组键"`
	LabelsDigest       string         `json:"labels_digest" gorm:"size:128;index;comment:标签摘要"`
	MatchedPolicyIDs   string         `json:"matched_policy_ids" gorm:"size:256;comment:命中策略ID列表,逗号分隔"`
	MatchedPolicyNames string         `json:"matched_policy_names" gorm:"size:512;comment:命中策略名称列表,逗号分隔"`
	MatchedPolicyIDList   []uint      `json:"matched_policy_id_list,omitempty" gorm:"-"`
	MatchedPolicyNameList []string    `json:"matched_policy_name_list,omitempty" gorm:"-"`
	ChannelID          uint           `json:"channel_id" gorm:"index;comment:通知渠道ID"`
	ChannelName        string         `json:"channel_name" gorm:"size:64;comment:通知渠道名称"`
	Success            bool           `json:"success" gorm:"not null;default:false;comment:通知是否成功"`
	HTTPStatusCode     int            `json:"http_status_code" gorm:"comment:通知响应状态码"`
	ErrorMessage       string         `json:"error_message" gorm:"size:1024;comment:错误信息"`
	RequestPayload     string         `json:"request_payload" gorm:"type:longtext;comment:请求载荷"`
	ResponsePayload    string         `json:"response_payload" gorm:"type:longtext;comment:响应载荷"`
	CreatedAt          time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt          time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 alert_events。
func (AlertEvent) TableName() string {
	return "alert_events"
}
