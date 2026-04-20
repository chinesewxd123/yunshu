package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type K8sServiceListQuery = ClusterNamespaceKeywordQuery
type K8sServiceDetailQuery = ClusterNamespaceNameQuery
type K8sServiceApplyRequest = ClusterManifestApplyRequest
type K8sServiceDeleteRequest = ClusterNamespaceNameQuery

type K8sServiceItem struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Type            string            `json:"type"`
	InternalTraffic string            `json:"internal_traffic,omitempty"`
	ClusterIP       string            `json:"cluster_ip,omitempty"`
	ExternalIPs     string            `json:"external_ips,omitempty"`
	Ports           string            `json:"ports,omitempty"`
	IPFamilies      string            `json:"ip_families,omitempty"`
	IPFamilyPolicy  string            `json:"ip_family_policy,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Selectors       map[string]string `json:"selectors,omitempty"`
	SelectorCount   int               `json:"selector_count"`
	SessionAffinity string            `json:"session_affinity,omitempty"`
	Age             string            `json:"age,omitempty"`
	CreationTime    string            `json:"creation_time"`
}

type K8sServiceDetail struct {
	YAML string `json:"yaml"`
}

type K8sServiceResourceService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sServiceResourceService 创建相关逻辑。
func NewK8sServiceResourceService(runtime *K8sRuntimeService) *K8sServiceResourceService {
	return &K8sServiceResourceService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var k8sServiceGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}

// List 查询列表相关的业务逻辑。
func (s *K8sServiceResourceService) List(ctx context.Context, q K8sServiceListQuery) ([]K8sServiceItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	if ns == "" {
		return nil, apperror.BadRequest("命名空间不能为空")
	}
	listU, err := s.dyn.ListByGVK(ctx, k, k8sServiceGVK, ns)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Service 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]K8sServiceItem, 0, len(listU))
	for _, u := range listU {
		var obj corev1.Service
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		externalIPs := "-"
		if len(obj.Spec.ExternalIPs) > 0 {
			externalIPs = strings.Join(obj.Spec.ExternalIPs, ",")
		} else if len(obj.Status.LoadBalancer.Ingress) > 0 {
			lb := make([]string, 0, len(obj.Status.LoadBalancer.Ingress))
			for _, ing := range obj.Status.LoadBalancer.Ingress {
				if ing.Hostname != "" {
					lb = append(lb, ing.Hostname)
				}
				if ing.IP != "" {
					lb = append(lb, ing.IP)
				}
			}
			if len(lb) > 0 {
				externalIPs = strings.Join(lb, ",")
			}
		}
		out = append(out, K8sServiceItem{
			Name:            obj.Name,
			Namespace:       obj.Namespace,
			Type:            string(obj.Spec.Type),
			InternalTraffic: fallbackDash(internalTrafficPolicyString(obj.Spec.InternalTrafficPolicy)),
			ClusterIP:       fallbackDash(obj.Spec.ClusterIP),
			ExternalIPs:     externalIPs,
			Ports:           joinServicePorts(obj.Spec.Ports),
			IPFamilies:      joinIPFamilies(obj.Spec.IPFamilies),
			IPFamilyPolicy:  fallbackDash(stringValue(obj.Spec.IPFamilyPolicy)),
			Labels:          obj.Labels,
			Annotations:     obj.Annotations,
			Selectors:       obj.Spec.Selector,
			SelectorCount:   len(obj.Spec.Selector),
			SessionAffinity: fallbackDash(string(obj.Spec.SessionAffinity)),
			Age:             k8sutil.HumanAge(obj.CreationTimestamp.Time),
			CreationTime:    obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sServiceResourceService) Detail(ctx context.Context, q K8sServiceDetailQuery) (*K8sServiceDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	if ns == "" {
		return nil, apperror.BadRequest("命名空间不能为空")
	}
	u, err := s.dyn.GetByGVK(ctx, k, k8sServiceGVK, ns, strings.TrimSpace(q.Name))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("Service 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Service 详情失败: %v", err))
	}
	obj := u.DeepCopy()
	obj.SetManagedFields(nil)
	y, _ := yaml.Marshal(obj.Object)
	return &K8sServiceDetail{YAML: string(y)}, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sServiceResourceService) Apply(ctx context.Context, req K8sServiceApplyRequest) error {
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

// Delete 删除相关的业务逻辑。
func (s *K8sServiceResourceService) Delete(ctx context.Context, req K8sServiceDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		return apperror.BadRequest("命名空间不能为空")
	}
	if err := s.dyn.DeleteByGVK(ctx, k, k8sServiceGVK, ns, strings.TrimSpace(req.Name)); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 Service 失败: %v", err))
	}
	return nil
}

func joinIPFamilies(f []corev1.IPFamily) string {
	if len(f) == 0 {
		return "-"
	}
	out := make([]string, 0, len(f))
	for _, x := range f {
		out = append(out, string(x))
	}
	return strings.Join(out, ",")
}

func stringValue(v *corev1.IPFamilyPolicyType) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func internalTrafficPolicyString(v *corev1.ServiceInternalTrafficPolicy) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func joinServicePorts(ports []corev1.ServicePort) string {
	if len(ports) == 0 {
		return "-"
	}
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		s := strconv.Itoa(int(p.Port))
		if p.Protocol != "" {
			s += "/" + string(p.Protocol)
		}
		if p.NodePort > 0 {
			s += "->" + strconv.Itoa(int(p.NodePort))
		}
		if p.TargetPort.String() != "" {
			s += "=>" + p.TargetPort.String()
		}
		out = append(out, s)
	}
	return strings.Join(out, ", ")
}
