package service

import (
	"context"
	"sort"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *K8sWorkloadService) DeploymentPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var d appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).Get(&d).Error; err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmta3018a66177e)
	}
	selector := metav1.FormatLabelSelector(d.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

// StatefulSetPods 执行对应的业务逻辑。
func (s *K8sWorkloadService) StatefulSetPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var st appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(q.Namespace).Name(q.Name).Get(&st).Error; err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt70dba6fa52bd)
	}
	selector := metav1.FormatLabelSelector(st.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

// DaemonSetPods 执行对应的业务逻辑。
func listPodsBySelector(ctx context.Context, k *kom.Kubectl, namespace, selector string) ([]RelatedPodItem, error) {
	if k == nil {
		return nil, constants.ErrInternalWithMsg(constants.ErrMsgc674e8a0802b)
	}
	opts := metav1.ListOptions{}
	if strings.TrimSpace(selector) != "" {
		opts.LabelSelector = strings.TrimSpace(selector)
	}
	var list []corev1.Pod
	query := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(namespace)
	if strings.TrimSpace(opts.LabelSelector) != "" {
		query = query.WithLabelSelector(strings.TrimSpace(opts.LabelSelector))
	}
	if err := query.List(&list).Error; err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt3ab38ee441a3)
	}
	out := make([]RelatedPodItem, 0, len(list))
	for _, p := range list {
		restarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		start := ""
		if p.Status.StartTime != nil && !p.Status.StartTime.IsZero() {
			start = p.Status.StartTime.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, RelatedPodItem{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Phase:        string(p.Status.Phase),
			NodeName:     p.Spec.NodeName,
			PodIP:        p.Status.PodIP,
			RestartCount: restarts,
			StartTime:    start,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// CronJobDetail 执行对应的业务逻辑。
