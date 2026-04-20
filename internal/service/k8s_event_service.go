package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/apperror"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventListQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Kind      string `form:"kind"`
	Name      string `form:"name"`
	Keyword   string `form:"keyword"`
	Limit     int64  `form:"limit"`
}

type EventItem struct {
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Count        int32  `json:"count"`
	FirstTime    string `json:"first_time,omitempty"`
	LastTime     string `json:"last_time,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	InvolvedKind string `json:"involved_kind,omitempty"`
	InvolvedName string `json:"involved_name,omitempty"`
}

type K8sEventService struct {
	runtime *K8sRuntimeService
}

// NewK8sEventService 创建相关逻辑。
func NewK8sEventService(runtime *K8sRuntimeService) *K8sEventService {
	return &K8sEventService{runtime: runtime}
}

// List 查询列表相关的业务逻辑。
func (s *K8sEventService) List(ctx context.Context, q EventListQuery) ([]EventItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	limit := q.Limit
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	var list []corev1.Event
	query := k.WithContext(ctx).Resource(&corev1.Event{}).Limit(int(limit))
	if ns == metav1.NamespaceAll {
		query = query.AllNamespace()
	} else {
		query = query.Namespace(ns)
	}
	if err := query.List(&list).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Events 失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	kind := strings.TrimSpace(q.Kind)
	name := strings.TrimSpace(q.Name)
	out := make([]EventItem, 0, len(list))
	for _, e := range list {
		if kind != "" && e.InvolvedObject.Kind != kind {
			continue
		}
		if name != "" && e.InvolvedObject.Name != name {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(e.Reason + " " + e.Message + " " + e.InvolvedObject.Name)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		first := ""
		last := ""
		if !e.FirstTimestamp.IsZero() {
			first = e.FirstTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		if !e.LastTimestamp.IsZero() {
			last = e.LastTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		creation := ""
		if !e.CreationTimestamp.IsZero() {
			creation = e.CreationTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, EventItem{
			Namespace:    e.Namespace,
			Type:         e.Type,
			Reason:       e.Reason,
			Message:      e.Message,
			Count:        e.Count,
			FirstTime:    first,
			LastTime:     last,
			CreationTime: creation,
			InvolvedKind: e.InvolvedObject.Kind,
			InvolvedName: e.InvolvedObject.Name,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastTime > out[j].LastTime
	})
	return out, nil
}
