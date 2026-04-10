package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type RbacListQuery struct {
	ClusterID  uint   `form:"cluster_id" binding:"required"`
	Namespace  string `form:"namespace"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	WithDetail bool   `form:"with_detail"`
}

type RbacNameQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Name      string `form:"name" binding:"required"`
}

type RbacApplyRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Manifest  string `json:"manifest" binding:"required"`
}

type RbacDeleteRequest struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Name      string `form:"name" binding:"required"`
}

type RoleListItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Rules        int    `json:"rules"`
	CreationTime string `json:"creation_time"`
}

type RoleBindingListItem struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	RoleRef      string   `json:"role_ref"`
	Subjects     []string `json:"subjects,omitempty"`
	CreationTime string   `json:"creation_time"`
}

type ClusterRoleListItem struct {
	Name         string `json:"name"`
	Rules        int    `json:"rules"`
	CreationTime string `json:"creation_time"`
}

type ClusterRoleBindingListItem struct {
	Name         string   `json:"name"`
	RoleRef      string   `json:"role_ref"`
	Subjects     []string `json:"subjects,omitempty"`
	CreationTime string   `json:"creation_time"`
}

type RbacDetail struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	// namespace 对 cluster-scoped 资源为空
	Namespace string `json:"namespace,omitempty"`
	YAML      string `json:"yaml"`
}

type K8sRBACService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sRBACService(runtime *K8sRuntimeService) *K8sRBACService {
	return &K8sRBACService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var (
	roleGVK              = schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}
	roleBindingGVK       = schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}
	clusterRoleGVK       = schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}
	clusterRoleBindingGVK = schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}
)

func (s *K8sRBACService) ListRoles(ctx context.Context, query RbacListQuery) ([]RoleListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(query.Namespace)
	if ns == "" {
		return nil, apperror.BadRequest("namespace 不能为空")
	}
	listU, err := s.dyn.ListByGVK(ctx, k, roleGVK, ns)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Role 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]RoleListItem, 0, len(listU))
	for _, u := range listU {
		var obj rbacv1.Role
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		out = append(out, RoleListItem{
			Name:         obj.Name,
			Namespace:    obj.Namespace,
			Rules:        len(obj.Rules),
			CreationTime: obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sRBACService) ListRoleBindings(ctx context.Context, query RbacListQuery) ([]RoleBindingListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(query.Namespace)
	if ns == "" {
		return nil, apperror.BadRequest("namespace 不能为空")
	}
	listU, err := s.dyn.ListByGVK(ctx, k, roleBindingGVK, ns)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 RoleBinding 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]RoleBindingListItem, 0, len(listU))
	for _, u := range listU {
		var obj rbacv1.RoleBinding
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		subjects := make([]string, 0, len(obj.Subjects))
		for _, s := range obj.Subjects {
			txt := strings.TrimSpace(s.Kind + ":" + s.Name)
			if s.Namespace != "" {
				txt += "@" + s.Namespace
			}
			subjects = append(subjects, txt)
		}
		sort.Strings(subjects)
		out = append(out, RoleBindingListItem{
			Name:         obj.Name,
			Namespace:    obj.Namespace,
			RoleRef:      strings.TrimSpace(obj.RoleRef.Kind + ":" + obj.RoleRef.Name),
			Subjects:     subjects,
			CreationTime: obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sRBACService) ListClusterRoles(ctx context.Context, query RbacListQuery) ([]ClusterRoleListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, clusterRoleGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 ClusterRole 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]ClusterRoleListItem, 0, len(listU))
	for _, u := range listU {
		var obj rbacv1.ClusterRole
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		out = append(out, ClusterRoleListItem{
			Name:         obj.Name,
			Rules:        len(obj.Rules),
			CreationTime: obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sRBACService) ListClusterRoleBindings(ctx context.Context, query RbacListQuery) ([]ClusterRoleBindingListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, clusterRoleBindingGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 ClusterRoleBinding 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]ClusterRoleBindingListItem, 0, len(listU))
	for _, u := range listU {
		var obj rbacv1.ClusterRoleBinding
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		subjects := make([]string, 0, len(obj.Subjects))
		for _, s := range obj.Subjects {
			txt := strings.TrimSpace(s.Kind + ":" + s.Name)
			if s.Namespace != "" {
				txt += "@" + s.Namespace
			}
			subjects = append(subjects, txt)
		}
		sort.Strings(subjects)
		out = append(out, ClusterRoleBindingListItem{
			Name:         obj.Name,
			RoleRef:      strings.TrimSpace(obj.RoleRef.Kind + ":" + obj.RoleRef.Name),
			Subjects:     subjects,
			CreationTime: obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sRBACService) Detail(ctx context.Context, kind string, query RbacNameQuery) (*RbacDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	kind = strings.TrimSpace(kind)
	name := strings.TrimSpace(query.Name)
	ns := strings.TrimSpace(query.Namespace)
	if kind == "" || name == "" {
		return nil, apperror.BadRequest("kind/name 不能为空")
	}

	var gvk schema.GroupVersionKind
	clusterScoped := false
	switch kind {
	case "Role":
		gvk = roleGVK
	case "RoleBinding":
		gvk = roleBindingGVK
	case "ClusterRole":
		gvk = clusterRoleGVK
		clusterScoped = true
	case "ClusterRoleBinding":
		gvk = clusterRoleBindingGVK
		clusterScoped = true
	default:
		return nil, apperror.BadRequest("不支持的 RBAC kind")
	}
	if !clusterScoped && ns == "" {
		return nil, apperror.BadRequest("namespace 不能为空")
	}

	u, err := s.dyn.GetByGVK(ctx, k, gvk, ns, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取详情失败: %v", err))
	}

	obj := u.DeepCopy()
	obj.SetManagedFields(nil)
	y, _ := yaml.Marshal(obj.Object)
	return &RbacDetail{Kind: kind, Name: name, Namespace: ns, YAML: string(y)}, nil
}

func (s *K8sRBACService) Apply(ctx context.Context, req RbacApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("manifest 不能为空")
	}
	// Apply 成功但返回 strings 的兼容逻辑已在 dyn.ApplyManifest 中处理
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		// 粗略判断：若 manifest 中提到了 rbac 资源且已存在，则视为成功
		refs := extractRbacRefs(req.Manifest)
		for _, ref := range refs {
			if _, e := s.dyn.GetByGVK(c, k, ref.GVK, ref.Namespace, ref.Name); e != nil {
				return false
			}
		}
		return len(refs) > 0
	}); err != nil {
		return apperror.Internal(fmt.Sprintf("应用 YAML 失败: %v", err))
	}
	return nil
}

func (s *K8sRBACService) Delete(ctx context.Context, kind string, req RbacDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	kind = strings.TrimSpace(kind)
	name := strings.TrimSpace(req.Name)
	ns := strings.TrimSpace(req.Namespace)
	var gvk schema.GroupVersionKind
	clusterScoped := false
	switch kind {
	case "Role":
		gvk = roleGVK
	case "RoleBinding":
		gvk = roleBindingGVK
	case "ClusterRole":
		gvk = clusterRoleGVK
		clusterScoped = true
	case "ClusterRoleBinding":
		gvk = clusterRoleBindingGVK
		clusterScoped = true
	default:
		return apperror.BadRequest("不支持的 RBAC kind")
	}
	if !clusterScoped && ns == "" {
		return apperror.BadRequest("namespace 不能为空")
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, ns, name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除失败: %v", err))
	}
	return nil
}

type rbacRef struct {
	GVK       schema.GroupVersionKind
	Namespace string
	Name      string
}

func extractRbacRefs(manifest string) []rbacRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]rbacRef, 0)
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
		apiVersion, _ := m["apiVersion"].(string)
		kind = strings.TrimSpace(kind)
		apiVersion = strings.TrimSpace(apiVersion)
		if !strings.HasPrefix(apiVersion, "rbac.authorization.k8s.io/") {
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
		if name == "" || kind == "" {
			continue
		}
		var gvk schema.GroupVersionKind
		switch kind {
		case "Role":
			gvk = roleGVK
		case "RoleBinding":
			gvk = roleBindingGVK
		case "ClusterRole":
			gvk = clusterRoleGVK
			ns = ""
		case "ClusterRoleBinding":
			gvk = clusterRoleBindingGVK
			ns = ""
		default:
			continue
		}
		out = append(out, rbacRef{GVK: gvk, Namespace: ns, Name: name})
	}
	return out
}

