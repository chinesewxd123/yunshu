package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
)

type OverviewResponse struct {
	UsersCount    int64 `json:"users_count"`
	ClustersCount int64 `json:"clusters_count"`

	PodNormalCount   int64 `json:"pod_normal_count"`
	PodAbnormalCount int64 `json:"pod_abnormal_count"`

	// Number of clusters that failed during pod aggregation.
	PodClusterErrors int64 `json:"pod_cluster_errors"`

	// Event stats (sampled per cluster to control latency).
	EventTotalCount    int64 `json:"event_total_count"`
	EventWarningCount  int64 `json:"event_warning_count"`
	EventClusterErrors int64 `json:"event_cluster_errors"`
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

	// 产品侧体验优先：总时限内返回“可得数据 + 失败计数”，而不是让首页等待到超时。
	overallCtx, overallCancel := context.WithTimeout(ctx, 8*time.Second)
	defer overallCancel()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, c := range clusters {
		cid := c.ID
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 连接探测保持短超时，避免不可达集群拖慢全局。
			cctx, cancel := context.WithTimeout(overallCtx, 2*time.Second)
			_, k, err := s.runtime.GetClusterKubectl(cctx, cid)
			cancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			// Pod 聚合也限制时长，超时按失败集群处理。
			pctx, pcancel := context.WithTimeout(overallCtx, 4*time.Second)
			var pods []corev1.Pod
			err = k.WithContext(pctx).Resource(&corev1.Pod{}).AllNamespace().List(&pods).Error
			pcancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			normal := int64(0)
			abnormal := int64(0)
			for _, p := range pods {
				if isPodNormal(p) {
					normal++
				} else {
					abnormal++
				}
			}
			mu.Lock()
			out.PodNormalCount += normal
			out.PodAbnormalCount += abnormal
			mu.Unlock()

			// Event 概览仅采样最近 500 条，避免在大集群拖慢首页。
			ectx, ecancel := context.WithTimeout(overallCtx, 4*time.Second)
			var events []corev1.Event
			err = k.WithContext(ectx).Resource(&corev1.Event{}).AllNamespace().Limit(500).List(&events).Error
			ecancel()
			if err != nil {
				mu.Lock()
				out.EventClusterErrors++
				mu.Unlock()
				return
			}
			total := int64(len(events))
			warnings := int64(0)
			for _, ev := range events {
				if ev.Type == "Warning" {
					warnings++
				}
			}
			mu.Lock()
			out.EventTotalCount += total
			out.EventWarningCount += warnings
			mu.Unlock()
		}()
	}
	wg.Wait()

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
	return fmt.Sprintf("users=%d clusters=%d pod_normal=%d pod_abnormal=%d pod_errors=%d event_total=%d event_warning=%d event_errors=%d",
		o.UsersCount, o.ClustersCount, o.PodNormalCount, o.PodAbnormalCount, o.PodClusterErrors, o.EventTotalCount, o.EventWarningCount, o.EventClusterErrors)
}
