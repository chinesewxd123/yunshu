package model

import "time"

// K8sClusterAccessGrant 集群维度的 K8s 访问档位（对齐 k8m：可授予角色 / 用户 / 用户组）。
// ClusterID=0 表示适用于全部已纳管集群。
type K8sClusterAccessGrant struct {
	ID            uint      `json:"id" gorm:"primaryKey;comment:主键"`
	PrincipalKind string    `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_k8s_cluster_grant;comment:role|user|group"`
	PrincipalRef  string    `json:"principal_ref" gorm:"size:128;not null;uniqueIndex:uk_k8s_cluster_grant;comment:角色码/用户ID字符串/组编码"`
	ClusterID     uint      `json:"cluster_id" gorm:"not null;default:0;uniqueIndex:uk_k8s_cluster_grant;comment:平台集群ID，0=全部集群"`
	Preset        string    `json:"preset" gorm:"size:32;not null;comment:readonly|readonly_exec|admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (K8sClusterAccessGrant) TableName() string { return "k8s_cluster_access_grants" }
