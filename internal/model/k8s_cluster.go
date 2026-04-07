package model

import (
	"time"

	"gorm.io/gorm"
)

// K8sCluster stores cluster connection configuration (kubeconfig) for Kom integration.
// NOTE: kubeconfig is sensitive; it's excluded from JSON responses.
type K8sCluster struct {
	ID uint `json:"id" gorm:"primaryKey"`

	Name string `json:"name" gorm:"size:128;not null;index"`

	// Kubeconfig is stored so the backend can register the cluster via Kom SDK.
	// Excluded from API responses; only used internally.
	Kubeconfig string `json:"-" gorm:"type:longtext;not null"`

	Status int `json:"status" gorm:"not null;default:1"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (K8sCluster) TableName() string {
	return "k8s_clusters"
}

