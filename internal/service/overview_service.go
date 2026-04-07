package service

import (
	"context"
	"fmt"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OverviewResponse struct {
	UsersCount    int64 `json:"users_count"`
	ClustersCount int64 `json:"clusters_count"`

	PodNormalCount   int64 `json:"pod_normal_count"`
	PodAbnormalCount int64 `json:"pod_abnormal_count"`

	// Number of clusters that failed during pod aggregation.
	PodClusterErrors int64 `json:"pod_cluster_errors"`
}

type OverviewService struct {
	db      *gorm.DB
	runtime *K8sRuntimeService
}

func NewOverviewService(db *gorm.DB, runtime *K8sRuntimeService) *OverviewService {
	return &OverviewService{db: db, runtime: runtime}
}

func (s *OverviewService) Get(ctx context.Context) (*OverviewResponse, error) {
	if s.db == nil {
		return nil, apperror.Internal("db not initialized")
	}

	out := &OverviewResponse{}
	if err := s.db.WithContext(ctx).Model(&model.User{}).Count(&out.UsersCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.K8sCluster{}).Count(&out.ClustersCount).Error; err != nil {
		return nil, err
	}

	// Pod stats: aggregate across enabled clusters.
	var clusters []model.K8sCluster
	if err := s.db.WithContext(ctx).Model(&model.K8sCluster{}).Where("status = ?", 1).Find(&clusters).Error; err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return out, nil
	}

	for _, c := range clusters {
		// keep per-cluster requests bounded
		cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
		_, k, err := s.runtime.GetClusterKubectl(cctx, c.ID)
		cancel()
		if err != nil {
			out.PodClusterErrors++
			continue
		}

		// list all pods in all namespaces
		pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
		pods, err := k.Client().CoreV1().Pods("").List(pctx, metav1.ListOptions{})
		pcancel()
		if err != nil {
			out.PodClusterErrors++
			continue
		}

		for _, p := range pods.Items {
			if isPodNormal(p) {
				out.PodNormalCount++
			} else {
				out.PodAbnormalCount++
			}
		}
	}

	return out, nil
}

func isPodNormal(p corev1.Pod) bool {
	// A pragmatic definition:
	// - phase is Running
	// - all containers are ready (or no container status found -> abnormal)
	if string(p.Status.Phase) != "Running" {
		return false
	}
	if len(p.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, st := range p.Status.ContainerStatuses {
		if !st.Ready {
			return false
		}
	}
	return true
}

func (o OverviewResponse) String() string {
	return fmt.Sprintf("users=%d clusters=%d normal=%d abnormal=%d errors=%d",
		o.UsersCount, o.ClustersCount, o.PodNormalCount, o.PodAbnormalCount, o.PodClusterErrors)
}
