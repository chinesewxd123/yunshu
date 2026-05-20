package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/pkg/k8sutil"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type HPAListQuery = ClusterNamespaceKeywordQuery
type HPADetailQuery = ClusterNamespaceNameQuery
type HPAApplyRequest = ClusterManifestApplyRequest
type HPADeleteRequest = ClusterNamespaceNameQuery

type HPAItem struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	MinReplicas    string            `json:"min_replicas,omitempty"`
	MaxReplicas    string            `json:"max_replicas,omitempty"`
	ScaleTargetRef string            `json:"scale_target_ref,omitempty"`
	MetricsSummary string            `json:"metrics_summary,omitempty"`
	ConditionsText string            `json:"conditions_text,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Age            string            `json:"age,omitempty"`
	CreationTime   string            `json:"creation_time"`
}

type HPADetail struct {
	YAML string `json:"yaml"`
}

type K8sHPAService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sHPAService(runtime *K8sRuntimeService) *K8sHPAService {
	return &K8sHPAService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var hpaGVK = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}

func (s *K8sHPAService) List(ctx context.Context, q HPAListQuery) ([]HPAItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, hpaGVK, q.Namespace)
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.hpa", "api", err, constants.ErrFmte5f4df2bc9c2)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]HPAItem, 0, len(listU))
	for _, item := range listU {
		var h autoscalingv2.HorizontalPodAutoscaler
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &h); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(h.Name), kw) {
			continue
		}
		out = append(out, hpaToItem(&h))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sHPAService) Detail(ctx context.Context, q HPADetailQuery) (*HPADetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, hpaGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsge64b05879667)
		}
		return nil, svcerr.Internal(ctx, "k8s.hpa", "api", err, constants.ErrFmtd28ea35ac553)
	}
	var obj autoscalingv2.HorizontalPodAutoscaler
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "autoscaling/v2"
	copyObj.Kind = "HorizontalPodAutoscaler"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &HPADetail{YAML: string(y)}, nil
}

func (s *K8sHPAService) Apply(ctx context.Context, req HPAApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, nil); err != nil {
		return k8sFail(ctx, "k8s.hpa", "api", err)
	}
	return nil
}

func (s *K8sHPAService) Delete(ctx context.Context, req HPADeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, hpaGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.hpa", "api", err)
	}
	return nil
}

func hpaToItem(h *autoscalingv2.HorizontalPodAutoscaler) HPAItem {
	min := "-"
	if h.Spec.MinReplicas != nil {
		min = strconv.Itoa(int(*h.Spec.MinReplicas))
	}
	max := strconv.Itoa(int(h.Spec.MaxReplicas))
	ref := ""
	if h.Spec.ScaleTargetRef.Name != "" {
		ref = strings.TrimSpace(h.Spec.ScaleTargetRef.Kind) + "/" + h.Spec.ScaleTargetRef.Name
	}
	metricsSummary := summarizeHPAMetrics(h.Spec.Metrics)
	condText := summarizeHPAConditions(h.Status.Conditions)
	return HPAItem{
		Name:           h.Name,
		Namespace:      h.Namespace,
		MinReplicas:    min,
		MaxReplicas:    max,
		ScaleTargetRef: ref,
		MetricsSummary: metricsSummary,
		ConditionsText: condText,
		Labels:         h.Labels,
		Age:            k8sutil.HumanAge(h.CreationTimestamp.Time),
		CreationTime:   h.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}

func summarizeHPAMetrics(metrics []autoscalingv2.MetricSpec) string {
	if len(metrics) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(metrics))
	for _, m := range metrics {
		switch m.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if m.Resource != nil && m.Resource.Name != "" {
				parts = append(parts, "Resource/"+string(m.Resource.Name))
			}
		case autoscalingv2.ContainerResourceMetricSourceType:
			if m.ContainerResource != nil && m.ContainerResource.Name != "" {
				parts = append(parts, "ContainerResource/"+string(m.ContainerResource.Name))
			}
		case autoscalingv2.PodsMetricSourceType:
			if m.Pods != nil && m.Pods.Metric.Name != "" {
				parts = append(parts, "Pods/"+m.Pods.Metric.Name)
			}
		case autoscalingv2.ObjectMetricSourceType:
			if m.Object != nil && m.Object.Metric.Name != "" {
				parts = append(parts, "Object/"+m.Object.Metric.Name)
			}
		case autoscalingv2.ExternalMetricSourceType:
			if m.External != nil && m.External.Metric.Name != "" {
				parts = append(parts, "External/"+m.External.Metric.Name)
			}
		default:
			parts = append(parts, string(m.Type))
		}
	}
	return strings.Join(parts, ",")
}

func summarizeHPAConditions(conds []autoscalingv2.HorizontalPodAutoscalerCondition) string {
	if len(conds) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(conds))
	for _, c := range conds {
		st := string(c.Status)
		parts = append(parts, fmt.Sprintf("%s=%s", c.Type, st))
	}
	return strings.Join(parts, "; ")
}
