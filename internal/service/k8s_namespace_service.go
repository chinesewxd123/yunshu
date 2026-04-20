package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type NamespaceListQuery = ClusterKeywordQuery
type NamespaceDetailQuery = ClusterNameQuery
type NamespaceApplyRequest = ClusterManifestApplyRequest
type NamespaceDeleteRequest = ClusterNameQuery

type NamespaceListItem struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	CreationTime string            `json:"creation_time"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`

	PodCount    int    `json:"pod_count"`
	CPURequests string `json:"cpu_requests,omitempty"`
	CPULimits   string `json:"cpu_limits,omitempty"`
	MemRequests string `json:"mem_requests,omitempty"`
	MemLimits   string `json:"mem_limits,omitempty"`
}

type NamespaceDetail struct {
	Item           NamespaceListItem         `json:"item"`
	Finalizers     []string                  `json:"finalizers,omitempty"`
	ResourceQuotas []NamespaceQuotaItem      `json:"resource_quotas,omitempty"`
	LimitRanges    []NamespaceLimitRangeItem `json:"limit_ranges,omitempty"`
	RecentEvents   []NamespaceEventItem      `json:"recent_events,omitempty"`
	YAML           string                    `json:"yaml"`
}

type NamespaceQuotaItem struct {
	Name  string            `json:"name"`
	Hard  map[string]string `json:"hard,omitempty"`
	Used  map[string]string `json:"used,omitempty"`
	Scope []string          `json:"scope,omitempty"`
}

type NamespaceLimitRangeItem struct {
	Name   string                  `json:"name"`
	Limits []corev1.LimitRangeItem `json:"limits,omitempty"`
}

type NamespaceEventItem struct {
	Type     string `json:"type"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	LastTime string `json:"last_time,omitempty"`
	Count    int32  `json:"count"`
}

type K8sNamespaceService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sNamespaceService 创建相关逻辑。
func NewK8sNamespaceService(runtime *K8sRuntimeService) *K8sNamespaceService {
	return &K8sNamespaceService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var namespaceGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}

// List 查询列表相关的业务逻辑。
func (s *K8sNamespaceService) List(ctx context.Context, query NamespaceListQuery) ([]NamespaceListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}

	podSummary := map[string]namespacePodSummary{}
	{
		var pods []corev1.Pod
		// 全量拉取一次，按 namespace 聚合（比对每个 namespace 分别 List 更省请求数）
		if e := k.WithContext(ctx).Resource(&corev1.Pod{}).List(&pods).Error; e == nil {
			for _, p := range pods {
				ns := strings.TrimSpace(p.Namespace)
				if ns == "" {
					continue
				}
				sum := podSummary[ns]
				sum.PodCount++
				for _, c := range p.Spec.Containers {
					if rq := c.Resources.Requests; rq != nil {
						sum.CPURequests.Add(rq[corev1.ResourceCPU])
						sum.MemRequests.Add(rq[corev1.ResourceMemory])
					}
					if lm := c.Resources.Limits; lm != nil {
						sum.CPULimits.Add(lm[corev1.ResourceCPU])
						sum.MemLimits.Add(lm[corev1.ResourceMemory])
					}
				}
				podSummary[ns] = sum
			}
		}
	}

	listU, err := s.dyn.ListByGVK(ctx, k, namespaceGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取命名空间失败: %v", err))
	}
	list := make([]corev1.Namespace, 0, len(listU))
	for _, item := range listU {
		var ns corev1.Namespace
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ns); e != nil {
			continue
		}
		list = append(list, ns)
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]NamespaceListItem, 0, len(list))
	for _, ns := range list {
		if kw != "" && !strings.Contains(strings.ToLower(ns.Name), kw) {
			continue
		}
		sum := podSummary[ns.Name]
		out = append(out, NamespaceListItem{
			Name:         ns.Name,
			Status:       string(ns.Status.Phase),
			CreationTime: ns.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Labels:       ns.Labels,
			Annotations:  ns.Annotations,
			PodCount:     sum.PodCount,
			CPURequests:  quantityOrDash(sum.CPURequests),
			CPULimits:    quantityOrDash(sum.CPULimits),
			MemRequests:  quantityOrDash(sum.MemRequests),
			MemLimits:    quantityOrDash(sum.MemLimits),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type namespacePodSummary struct {
	PodCount    int
	CPURequests resource.Quantity
	CPULimits   resource.Quantity
	MemRequests resource.Quantity
	MemLimits   resource.Quantity
}

func quantityOrDash(q resource.Quantity) string {
	if q.IsZero() {
		return "-"
	}
	return q.String()
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sNamespaceService) Detail(ctx context.Context, query NamespaceDetailQuery) (*NamespaceDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, namespaceGVK, "", query.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("命名空间不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取命名空间详情失败: %v", err))
	}
	var ns corev1.Namespace
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ns)
	copyObj := ns.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "Namespace"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	quotaItems, _ := s.listNamespaceQuotas(ctx, k, query.Name)
	limitItems, _ := s.listNamespaceLimitRanges(ctx, k, query.Name)
	eventItems, _ := s.listNamespaceEvents(ctx, k, query.Name)
	finalizers := make([]string, 0, len(copyObj.Spec.Finalizers))
	for _, f := range copyObj.Spec.Finalizers {
		finalizers = append(finalizers, string(f))
	}
	return &NamespaceDetail{
		Item: NamespaceListItem{
			Name:         copyObj.Name,
			Status:       string(copyObj.Status.Phase),
			CreationTime: copyObj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Labels:       copyObj.Labels,
			Annotations:  copyObj.Annotations,
		},
		Finalizers:     finalizers,
		ResourceQuotas: quotaItems,
		LimitRanges:    limitItems,
		RecentEvents:   eventItems,
		YAML:           string(y),
	}, nil
}

func mapQuantityToString(v corev1.ResourceList) map[string]string {
	out := make(map[string]string, len(v))
	for k, q := range v {
		out[string(k)] = q.String()
	}
	return out
}

func (s *K8sNamespaceService) listNamespaceQuotas(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceQuotaItem, error) {
	var list []corev1.ResourceQuota
	if err := k.WithContext(ctx).Resource(&corev1.ResourceQuota{}).Namespace(namespace).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceQuotaItem, 0, len(list))
	for _, q := range list {
		scope := make([]string, 0, len(q.Spec.Scopes))
		for _, s := range q.Spec.Scopes {
			scope = append(scope, string(s))
		}
		out = append(out, NamespaceQuotaItem{
			Name:  q.Name,
			Hard:  mapQuantityToString(q.Status.Hard),
			Used:  mapQuantityToString(q.Status.Used),
			Scope: scope,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sNamespaceService) listNamespaceLimitRanges(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceLimitRangeItem, error) {
	var list []corev1.LimitRange
	if err := k.WithContext(ctx).Resource(&corev1.LimitRange{}).Namespace(namespace).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceLimitRangeItem, 0, len(list))
	for _, r := range list {
		out = append(out, NamespaceLimitRangeItem{
			Name:   r.Name,
			Limits: r.Spec.Limits,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sNamespaceService) listNamespaceEvents(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceEventItem, error) {
	var list []corev1.Event
	if err := k.WithContext(ctx).Resource(&corev1.Event{}).Namespace(namespace).Limit(20).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceEventItem, 0, len(list))
	for _, e := range list {
		lastTime := ""
		if !e.LastTimestamp.IsZero() {
			lastTime = e.LastTimestamp.Time.Format("2006-01-02 15:04:05")
		} else if !e.CreationTimestamp.IsZero() {
			lastTime = e.CreationTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, NamespaceEventItem{
			Type:     e.Type,
			Reason:   e.Reason,
			Message:  e.Message,
			LastTime: lastTime,
			Count:    e.Count,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastTime > out[j].LastTime })
	if len(out) > 10 {
		out = out[:10]
	}
	return out, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sNamespaceService) Apply(ctx context.Context, req NamespaceApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("资源清单不能为空")
	}
	refs := extractNamespaceRefs(req.Manifest)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, name := range refs {
			if _, e := s.dyn.GetByGVK(c, k, namespaceGVK, "", name); e != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return apperror.Internal(fmt.Sprintf("应用 YAML 失败: %v", err))
	}
	return nil
}

func extractNamespaceRefs(manifest string) []string {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]string, 0)
	for _, doc := range docs {
		docTrim := strings.TrimSpace(doc)
		if docTrim == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(docTrim), &m); err != nil {
			continue
		}
		kind, _ := m["kind"].(string)
		if strings.TrimSpace(kind) != "Namespace" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Delete 删除相关的业务逻辑。
func (s *K8sNamespaceService) Delete(ctx context.Context, req NamespaceDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, namespaceGVK, "", req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除命名空间失败: %v", err))
	}
	return nil
}
