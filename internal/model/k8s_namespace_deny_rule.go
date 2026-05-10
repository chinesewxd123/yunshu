package model

import "time"

// K8sNamespaceDenyRule 命名空间黑名单（对齐 k8m；黑名单优先于白名单与档位放行）。
type K8sNamespaceDenyRule struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	PrincipalKind string    `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_k8s_ns_deny;comment:role|user|group"`
	PrincipalRef  string    `json:"principal_ref" gorm:"size:128;not null;uniqueIndex:uk_k8s_ns_deny;comment:角色码/用户ID/组编码"`
	ClusterID     uint      `json:"cluster_id" gorm:"not null;uniqueIndex:uk_k8s_ns_deny;comment:集群ID"`
	Namespace     string    `json:"namespace" gorm:"size:253;not null;uniqueIndex:uk_k8s_ns_deny;comment:禁止的命名空间名称"`
	CreatedAt     time.Time `json:"created_at" gorm:"comment:创建时间"`
}

func (K8sNamespaceDenyRule) TableName() string { return "k8s_namespace_deny_rules" }
