package model

import (
	"time"

	"gorm.io/gorm"
)

// K8sCluster 已接入的 Kubernetes 集群：名称与 kubeconfig（仅服务端存储，不落 JSON）。
type K8sCluster struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name string `json:"name" gorm:"size:128;not null;index;comment:集群名称"`

	// ConnectionMode 连接模式: kubeconfig 或 direct
	ConnectionMode string `json:"-" gorm:"size:32;default:'kubeconfig';comment:连接模式 kubeconfig/direct"`

	// Kubeconfig is stored so the backend can register the cluster via Kom SDK.
	// Excluded from API responses; only used internally.
	Kubeconfig string `json:"-" gorm:"type:longtext;not null;comment:集群连接配置"`

	// DirectConfig 直连配置 JSON，当 ConnectionMode=direct 时使用
	DirectConfig string `json:"-" gorm:"type:longtext;comment:直连配置JSON"`

	Status int `json:"status" gorm:"not null;default:1;index;comment:状态 1启用 0禁用"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 k8s_clusters。
func (K8sCluster) TableName() string {
	return "k8s_clusters"
}
