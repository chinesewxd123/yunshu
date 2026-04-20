package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/k8sutil"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type NetworkPolicyListQuery = ClusterNamespaceKeywordQuery
type NetworkPolicyDetailQuery = ClusterNamespaceNameQuery
type NetworkPolicyApplyRequest = ClusterManifestApplyRequest
type NetworkPolicyDeleteRequest = ClusterNamespaceNameQuery

type NetworkPolicyItem struct {
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	PolicyTypes      string            `json:"policy_types,omitempty"`
	PodSelectorCount int               `json:"pod_selector_count"`
	IngressRuleCount int               `json:"ingress_rule_count"`
	EgressRuleCount  int               `json:"egress_rule_count"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	Age              string            `json:"age,omitempty"`
	CreationTime     string            `json:"creation_time"`
}

type NetworkPolicyDetail struct {
	YAML string `json:"yaml"`
}

type K8sNetworkPolicyService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sNetworkPolicyService(runtime *K8sRuntimeService) *K8sNetworkPolicyService {
	return &K8sNetworkPolicyService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var networkPolicyGVK = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}

func (s *K8sNetworkPolicyService) List(ctx context.Context, q NetworkPolicyListQuery) ([]NetworkPolicyItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, networkPolicyGVK, q.Namespace)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 NetworkPolicy 列表失败: %v", err))
	}

	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]NetworkPolicyItem, 0, len(listU))
	for _, item := range listU {
		var np networkingv1.NetworkPolicy
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &np); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(np.Name), kw) {
			continue
		}
		out = append(out, networkPolicyToItem(&np))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sNetworkPolicyService) Detail(ctx context.Context, q NetworkPolicyDetailQuery) (*NetworkPolicyDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, networkPolicyGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("NetworkPolicy 资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 NetworkPolicy 详情失败: %v", err))
	}
	var obj networkingv1.NetworkPolicy
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "networking.k8s.io/v1"
	copyObj.Kind = "NetworkPolicy"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &NetworkPolicyDetail{YAML: string(y)}, nil
}

func (s *K8sNetworkPolicyService) Apply(ctx context.Context, req NetworkPolicyApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("资源清单不能为空")
	}
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, nil); err != nil {
		return apperror.Internal(fmt.Sprintf("应用 YAML 失败: %v", err))
	}
	return nil
}

func (s *K8sNetworkPolicyService) Delete(ctx context.Context, req NetworkPolicyDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, networkPolicyGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 NetworkPolicy 失败: %v", err))
	}
	return nil
}

func networkPolicyToItem(np *networkingv1.NetworkPolicy) NetworkPolicyItem {
	return NetworkPolicyItem{
		Name:             np.Name,
		Namespace:        np.Namespace,
		PolicyTypes:      joinPolicyTypes(np.Spec.PolicyTypes),
		PodSelectorCount: len(np.Spec.PodSelector.MatchLabels),
		IngressRuleCount: len(np.Spec.Ingress),
		EgressRuleCount:  len(np.Spec.Egress),
		Labels:           np.Labels,
		Annotations:      np.Annotations,
		Age:              k8sutil.HumanAge(np.CreationTimestamp.Time),
		CreationTime:     np.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}

func joinPolicyTypes(types []networkingv1.PolicyType) string {
	if len(types) == 0 {
		return "-"
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	return strings.Join(out, ",")
}
