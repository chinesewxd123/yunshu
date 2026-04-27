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
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type ServiceAccountListQuery = ClusterNamespaceKeywordQuery
type ServiceAccountDetailQuery = ClusterNamespaceNameQuery
type ServiceAccountApplyRequest = ClusterManifestApplyRequest
type ServiceAccountDeleteRequest = ClusterNamespaceNameQuery

type ServiceAccountListItem struct {
	Name                 string `json:"name"`
	Namespace            string `json:"namespace"`
	SecretsCount         int    `json:"secrets_count"`
	ImagePullSecretsCount int   `json:"image_pull_secrets_count"`
	CreationTime         string `json:"creation_time"`
}

type ServiceAccountBindingRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	RoleRef   string `json:"role_ref"`
}

type ServiceAccountDetail struct {
	YAML                string                     `json:"yaml"`
	Secrets             []string                   `json:"secrets,omitempty"`
	ImagePullSecrets    []string                   `json:"image_pull_secrets,omitempty"`
	RoleBindings        []ServiceAccountBindingRef `json:"role_bindings,omitempty"`
	ClusterRoleBindings []ServiceAccountBindingRef `json:"cluster_role_bindings,omitempty"`
}

type K8sServiceAccountService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sServiceAccountService(runtime *K8sRuntimeService) *K8sServiceAccountService {
	return &K8sServiceAccountService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var (
	serviceAccountGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
)

func (s *K8sServiceAccountService) List(ctx context.Context, q ServiceAccountListQuery) ([]ServiceAccountListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, serviceAccountGVK, q.Namespace)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 ServiceAccount 列表失败: %v", err))
	}

	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]ServiceAccountListItem, 0, len(listU))
	for _, u := range listU {
		var obj corev1.ServiceAccount
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		out = append(out, ServiceAccountListItem{
			Name:                  obj.Name,
			Namespace:             obj.Namespace,
			SecretsCount:          len(obj.Secrets),
			ImagePullSecretsCount: len(obj.ImagePullSecrets),
			CreationTime:          obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *K8sServiceAccountService) Detail(ctx context.Context, q ServiceAccountDetailQuery) (*ServiceAccountDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}

	u, err := s.dyn.GetByGVK(ctx, k, serviceAccountGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("ServiceAccount 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 ServiceAccount 详情失败: %v", err))
	}
	var obj corev1.ServiceAccount
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "ServiceAccount"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)

	secrets := make([]string, 0, len(copyObj.Secrets))
	for _, it := range copyObj.Secrets {
		if strings.TrimSpace(it.Name) != "" {
			secrets = append(secrets, strings.TrimSpace(it.Name))
		}
	}
	sort.Strings(secrets)

	imagePullSecrets := make([]string, 0, len(copyObj.ImagePullSecrets))
	for _, it := range copyObj.ImagePullSecrets {
		if strings.TrimSpace(it.Name) != "" {
			imagePullSecrets = append(imagePullSecrets, strings.TrimSpace(it.Name))
		}
	}
	sort.Strings(imagePullSecrets)

	roleBindings, clusterRoleBindings, err := s.collectBindings(ctx, k, q.Namespace, q.Name)
	if err != nil {
		return nil, err
	}

	return &ServiceAccountDetail{
		YAML:                string(y),
		Secrets:             secrets,
		ImagePullSecrets:    imagePullSecrets,
		RoleBindings:        roleBindings,
		ClusterRoleBindings: clusterRoleBindings,
	}, nil
}

func (s *K8sServiceAccountService) Apply(ctx context.Context, req ServiceAccountApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("资源清单不能为空")
	}

	refs := extractServiceAccountRefs(req.Manifest)
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		return serviceAccountRefsAllExist(c, s.dyn, k, refs)
	}); err != nil {
		return apperror.Internal(fmt.Sprintf("应用 YAML 失败: %v", err))
	}
	return nil
}

func (s *K8sServiceAccountService) Delete(ctx context.Context, req ServiceAccountDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, serviceAccountGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 ServiceAccount 失败: %v", err))
	}
	return nil
}

func (s *K8sServiceAccountService) collectBindings(ctx context.Context, k *kom.Kubectl, namespace, saName string) ([]ServiceAccountBindingRef, []ServiceAccountBindingRef, error) {
	roleBindings := make([]ServiceAccountBindingRef, 0)
	clusterRoleBindings := make([]ServiceAccountBindingRef, 0)

	rbList, err := s.dyn.ListByGVK(ctx, k, roleBindingGVK, namespace)
	if err != nil {
		return nil, nil, apperror.Internal(fmt.Sprintf("获取 RoleBinding 列表失败: %v", err))
	}
	for _, u := range rbList {
		var rb rbacv1.RoleBinding
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &rb); e != nil {
			continue
		}
		if !containsSARef(rb.Subjects, namespace, saName) {
			continue
		}
		roleBindings = append(roleBindings, ServiceAccountBindingRef{
			Name:      rb.Name,
			Namespace: rb.Namespace,
			RoleRef:   strings.TrimSpace(rb.RoleRef.Kind + ":" + rb.RoleRef.Name),
		})
	}
	sort.Slice(roleBindings, func(i, j int) bool {
		if roleBindings[i].Namespace != roleBindings[j].Namespace {
			return roleBindings[i].Namespace < roleBindings[j].Namespace
		}
		return roleBindings[i].Name < roleBindings[j].Name
	})

	crbList, err := s.dyn.ListByGVK(ctx, k, clusterRoleBindingGVK, "")
	if err != nil {
		return nil, nil, apperror.Internal(fmt.Sprintf("获取 ClusterRoleBinding 列表失败: %v", err))
	}
	for _, u := range crbList {
		var crb rbacv1.ClusterRoleBinding
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crb); e != nil {
			continue
		}
		if !containsSARef(crb.Subjects, namespace, saName) {
			continue
		}
		clusterRoleBindings = append(clusterRoleBindings, ServiceAccountBindingRef{
			Name:    crb.Name,
			RoleRef: strings.TrimSpace(crb.RoleRef.Kind + ":" + crb.RoleRef.Name),
		})
	}
	sort.Slice(clusterRoleBindings, func(i, j int) bool { return clusterRoleBindings[i].Name < clusterRoleBindings[j].Name })
	return roleBindings, clusterRoleBindings, nil
}

func containsSARef(subjects []rbacv1.Subject, namespace, name string) bool {
	ns := strings.TrimSpace(namespace)
	n := strings.TrimSpace(name)
	for _, sb := range subjects {
		if sb.Kind != "ServiceAccount" {
			continue
		}
		if strings.TrimSpace(sb.Name) != n {
			continue
		}
		subjectNS := strings.TrimSpace(sb.Namespace)
		if subjectNS == "" || subjectNS == ns {
			return true
		}
	}
	return false
}

type serviceAccountRef struct {
	Namespace string
	Name      string
}

func extractServiceAccountRefs(manifest string) []serviceAccountRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]serviceAccountRef, 0)
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
		if strings.TrimSpace(kind) != "ServiceAccount" || strings.TrimSpace(apiVersion) != "v1" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(meta["name"]))
		if name == "" {
			continue
		}
		namespace := strings.TrimSpace(fmt.Sprint(meta["namespace"]))
		if namespace == "" {
			namespace = "default"
		}
		out = append(out, serviceAccountRef{Namespace: namespace, Name: name})
	}
	return out
}

func serviceAccountRefsAllExist(ctx context.Context, dyn *DynamicResourceService, k *kom.Kubectl, refs []serviceAccountRef) bool {
	if dyn == nil || k == nil || len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		if _, err := dyn.GetByGVK(ctx, k, serviceAccountGVK, ref.Namespace, ref.Name); err != nil {
			return false
		}
	}
	return true
}
