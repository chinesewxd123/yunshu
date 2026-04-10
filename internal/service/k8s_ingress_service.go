package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type IngressListQuery = ClusterNamespaceKeywordQuery
type IngressDetailQuery = ClusterNamespaceNameQuery
type IngressApplyRequest = ClusterManifestApplyRequest
type IngressDeleteRequest = ClusterNamespaceNameQuery
type IngressClassListQuery = ClusterKeywordQuery
type IngressClassDetailQuery = ClusterNameQuery
type IngressClassApplyRequest = ClusterManifestApplyRequest
type IngressClassDeleteRequest = ClusterNameQuery

type IngressNginxRestartRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace"`
	Selector  string `json:"selector"`
}

type IngressNginxRestartResult struct {
	DeletedCount int      `json:"deleted_count"`
	DeletedNames []string `json:"deleted_names"`
}

type IngressItem struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	ClassName    string            `json:"class_name,omitempty"`
	RulesText    string            `json:"rules_text,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	HostCount    int               `json:"host_count"`
	TLSCount     int               `json:"tls_count"`
	LoadBalancer string            `json:"load_balancer,omitempty"`
	Age          string            `json:"age,omitempty"`
	CreationTime string            `json:"creation_time"`
}

type IngressClassItem struct {
	Name         string            `json:"name"`
	Controller   string            `json:"controller,omitempty"`
	IngressCount int               `json:"ingress_count"`
	IsDefault    bool              `json:"is_default"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	Age          string            `json:"age,omitempty"`
	CreationTime string            `json:"creation_time"`
}

type IngressClassDetail struct {
	Item IngressClassItem `json:"item"`
	YAML string           `json:"yaml"`
}

type IngressDetail struct {
	Item IngressItem `json:"item"`
	YAML string      `json:"yaml"`
}

type K8sIngressService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sIngressService(runtime *K8sRuntimeService) *K8sIngressService {
	return &K8sIngressService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var ingressGVK = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}
var ingressClassGVK = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"}

func (s *K8sIngressService) List(ctx context.Context, q IngressListQuery) ([]IngressItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, ingressGVK, q.Namespace)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Ingress 列表失败: %v", err))
	}
	list := make([]networkingv1.Ingress, 0, len(listU))
	for _, item := range listU {
		var ing networkingv1.Ingress
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ing); e != nil {
			continue
		}
		list = append(list, ing)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]IngressItem, 0, len(list))
	for _, ing := range list {
		if kw != "" && !strings.Contains(strings.ToLower(ing.Name), kw) {
			continue
		}
		out = append(out, ingressToItem(&ing))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sIngressService) Detail(ctx context.Context, q IngressDetailQuery) (*IngressDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, ingressGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("Ingress 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Ingress 详情失败: %v", err))
	}
	var obj networkingv1.Ingress
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "networking.k8s.io/v1"
	copyObj.Kind = "Ingress"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &IngressDetail{Item: ingressToItem(copyObj), YAML: string(y)}, nil
}

func (s *K8sIngressService) Apply(ctx context.Context, req IngressApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("manifest 不能为空")
	}
	refs := extractIngressRefs(req.Manifest)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, r := range refs {
			if strings.TrimSpace(r.Name) == "" {
				continue
			}
			ns := strings.TrimSpace(r.Namespace)
			if ns == "" {
				ns = "default"
			}
			if _, e := s.dyn.GetByGVK(c, k, ingressGVK, ns, r.Name); e != nil {
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

func (s *K8sIngressService) Delete(ctx context.Context, req IngressDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, ingressGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 Ingress 失败: %v", err))
	}
	return nil
}

func (s *K8sIngressService) ListClasses(ctx context.Context, q IngressClassListQuery) ([]IngressClassItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, ingressClassGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 IngressClass 列表失败: %v", err))
	}
	ingsU, err := s.dyn.ListByGVK(ctx, k, ingressGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Ingress 列表失败: %v", err))
	}
	classCounter := map[string]int{}
	for _, item := range ingsU {
		var ing networkingv1.Ingress
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ing); e != nil {
			continue
		}
		className := ""
		if ing.Spec.IngressClassName != nil {
			className = strings.TrimSpace(*ing.Spec.IngressClassName)
		}
		if className == "" {
			continue
		}
		classCounter[className]++
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]IngressClassItem, 0, len(listU))
	for _, item := range listU {
		var cls networkingv1.IngressClass
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cls); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(cls.Name), kw) {
			continue
		}
		out = append(out, ingressClassToItem(&cls, classCounter[cls.Name]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sIngressService) DetailClass(ctx context.Context, q IngressClassDetailQuery) (*IngressClassDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, ingressClassGVK, "", q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("IngressClass 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 IngressClass 详情失败: %v", err))
	}
	var obj networkingv1.IngressClass
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "networking.k8s.io/v1"
	copyObj.Kind = "IngressClass"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &IngressClassDetail{Item: ingressClassToItem(copyObj, 0), YAML: string(y)}, nil
}

func (s *K8sIngressService) ApplyClass(ctx context.Context, req IngressClassApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("manifest 不能为空")
	}
	refs := extractIngressClassRefs(req.Manifest)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, name := range refs {
			if strings.TrimSpace(name) == "" {
				continue
			}
			if _, e := s.dyn.GetByGVK(c, k, ingressClassGVK, "", name); e != nil {
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

func (s *K8sIngressService) DeleteClass(ctx context.Context, req IngressClassDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, ingressClassGVK, "", req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 IngressClass 失败: %v", err))
	}
	return nil
}

// RestartIngressNginxPods 删除 ingress-nginx controller Pods，使其自动重建，从而刷新默认证书等运行态资源。
// 说明：不同安装方式 label 可能不同，这里支持自定义 selector，并提供默认 selector。
func (s *K8sIngressService) RestartIngressNginxPods(ctx context.Context, req IngressNginxRestartRequest) (*IngressNginxRestartResult, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = "ingress-nginx"
	}
	selector := strings.TrimSpace(req.Selector)
	if selector == "" {
		// 官方 chart/controller 常见标签
		selector = "app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller"
	}

	var pods []corev1.Pod
	q := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).WithLabelSelector(selector)
	if err := q.List(&pods).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 ingress-nginx Pods 失败: %v", err))
	}
	// 兜底：如果 selector 太严格导致空，再尝试兼容历史 label
	if len(pods) == 0 && strings.TrimSpace(req.Selector) == "" {
		fallback := "app=ingress-nginx"
		_ = k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).WithLabelSelector(fallback).List(&pods).Error
	}
	if len(pods) == 0 {
		return &IngressNginxRestartResult{DeletedCount: 0, DeletedNames: []string{}}, nil
	}

	deleted := make([]string, 0, len(pods))
	for _, p := range pods {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue
		}
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).Name(name).Delete().Error; err != nil {
			// 删除部分失败也尽量继续，避免“一颗老鼠屎坏一锅汤”
			continue
		}
		deleted = append(deleted, name)
	}
	sort.Strings(deleted)
	return &IngressNginxRestartResult{DeletedCount: len(deleted), DeletedNames: deleted}, nil
}

type ingressRef struct {
	Name      string
	Namespace string
}

func extractIngressRefs(manifest string) []ingressRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]ingressRef, 0)
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
		if strings.TrimSpace(kind) != "Ingress" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		name = strings.TrimSpace(name)
		ns = strings.TrimSpace(ns)
		if name != "" && ns != "" {
			out = append(out, ingressRef{Name: name, Namespace: ns})
		}
	}
	return out
}

func extractIngressClassRefs(manifest string) []string {
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
		if strings.TrimSpace(kind) != "IngressClass" {
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

func ingressToItem(ing *networkingv1.Ingress) IngressItem {
	className := ""
	if ing.Spec.IngressClassName != nil {
		className = strings.TrimSpace(*ing.Spec.IngressClassName)
	}
	hostSet := map[string]bool{}
	for _, r := range ing.Spec.Rules {
		h := strings.TrimSpace(r.Host)
		if h != "" {
			hostSet[h] = true
		}
	}
	lb := ""
	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		first := ing.Status.LoadBalancer.Ingress[0]
		if strings.TrimSpace(first.Hostname) != "" {
			lb = first.Hostname
		} else {
			lb = first.IP
		}
	}
	rules := make([]string, 0, len(ing.Spec.Rules))
	for _, r := range ing.Spec.Rules {
		host := strings.TrimSpace(r.Host)
		if host == "" {
			host = "*"
		}
		if r.HTTP == nil || len(r.HTTP.Paths) == 0 {
			rules = append(rules, fmt.Sprintf("%s -> -", host))
			continue
		}
		for _, p := range r.HTTP.Paths {
			path := strings.TrimSpace(p.Path)
			if path == "" {
				path = "/"
			}
			svc := ""
			if p.Backend.Service != nil {
				svc = strings.TrimSpace(p.Backend.Service.Name)
				if p.Backend.Service.Port.Number > 0 {
					svc = fmt.Sprintf("%s:%d", svc, p.Backend.Service.Port.Number)
				} else if strings.TrimSpace(p.Backend.Service.Port.Name) != "" {
					svc = fmt.Sprintf("%s:%s", svc, strings.TrimSpace(p.Backend.Service.Port.Name))
				}
			}
			if svc == "" {
				svc = "-"
			}
			rules = append(rules, fmt.Sprintf("%s%s -> %s", host, path, svc))
		}
	}
	rulesText := strings.Join(rules, "\n")
	if strings.TrimSpace(rulesText) == "" {
		rulesText = "-"
	}
	return IngressItem{
		Name:         ing.Name,
		Namespace:    ing.Namespace,
		ClassName:    className,
		RulesText:    rulesText,
		Labels:       ing.Labels,
		Annotations:  ing.Annotations,
		HostCount:    len(hostSet),
		TLSCount:     len(ing.Spec.TLS),
		LoadBalancer: lb,
		Age:          k8sutil.HumanAge(ing.CreationTimestamp.Time),
		CreationTime: ing.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}

func ingressClassToItem(cls *networkingv1.IngressClass, ingressCount int) IngressClassItem {
	isDefault := false
	if cls.Annotations != nil {
		v := strings.TrimSpace(cls.Annotations["ingressclass.kubernetes.io/is-default-class"])
		isDefault = strings.EqualFold(v, "true")
	}
	return IngressClassItem{
		Name:         cls.Name,
		Controller:   strings.TrimSpace(cls.Spec.Controller),
		IngressCount: ingressCount,
		IsDefault:    isDefault,
		Labels:       cls.Labels,
		Annotations:  cls.Annotations,
		Age:          k8sutil.HumanAge(cls.CreationTimestamp.Time),
		CreationTime: cls.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}
