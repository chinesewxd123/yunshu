package model

import "time"

// K8sNamespaceAllowRule 命名空间白名单（对齐 k8m）：若某主体在某集群下存在任意允许规则，则仅允许访问规则中出现的命名空间（黑名单仍优先）。
type K8sNamespaceAllowRule struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	PrincipalKind string    `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_k8s_ns_allow;comment:role|user|group"`
	PrincipalRef  string    `json:"principal_ref" gorm:"size:128;not null;uniqueIndex:uk_k8s_ns_allow;comment:角色码/用户ID/组编码"`
	ClusterID     uint      `json:"cluster_id" gorm:"not null;uniqueIndex:uk_k8s_ns_allow;comment:集群ID"`
	Namespace     string    `json:"namespace" gorm:"size:253;not null;uniqueIndex:uk_k8s_ns_allow;comment:允许的命名空间"`
	CreatedAt     time.Time `json:"created_at"`
}

func (K8sNamespaceAllowRule) TableName() string { return "k8s_namespace_allow_rules" }
