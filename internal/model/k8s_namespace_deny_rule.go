package model

import "time"

// K8sNamespaceDenyRule 按「角色码 + 集群」禁止访问指定命名空间（对齐 k8m 黑名单语义；黑名单优先于三元放行）。
type K8sNamespaceDenyRule struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RoleCode  string    `json:"role_code" gorm:"size:64;not null;uniqueIndex:uk_k8s_ns_deny,priority:1;comment:角色编码"`
	ClusterID uint      `json:"cluster_id" gorm:"not null;uniqueIndex:uk_k8s_ns_deny,priority:2;comment:集群ID"`
	Namespace string    `json:"namespace" gorm:"size:253;not null;uniqueIndex:uk_k8s_ns_deny,priority:3;comment:禁止的命名空间名称"`
	CreatedAt time.Time `json:"created_at" gorm:"comment:创建时间"`
}

// TableName 表名。
func (K8sNamespaceDenyRule) TableName() string { return "k8s_namespace_deny_rules" }
