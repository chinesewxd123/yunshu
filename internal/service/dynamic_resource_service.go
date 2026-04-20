package service

import (
	"context"
	"fmt"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type DynamicResourceService struct {
	runtime *K8sRuntimeService
}

// NewDynamicResourceService 创建相关逻辑。
func NewDynamicResourceService(runtime *K8sRuntimeService) *DynamicResourceService {
	return &DynamicResourceService{runtime: runtime}
}

// ListByGVK 查询列表相关的业务逻辑。
func (s *DynamicResourceService) ListByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace string) ([]unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	var list []unstructured.Unstructured
	q := k.WithContext(ctx).Resource(u)
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListByGVKWithSelector 查询列表相关的业务逻辑。
func (s *DynamicResourceService) ListByGVKWithSelector(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, labelSelector string) ([]unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	var list []unstructured.Unstructured
	q := k.WithContext(ctx).Resource(u)
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	if ls := strings.TrimSpace(labelSelector); ls != "" {
		q = q.WithLabelSelector(ls)
	}
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetByGVK 获取相关的业务逻辑。
func (s *DynamicResourceService) GetByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, name string) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	q := k.WithContext(ctx).Resource(u).Name(strings.TrimSpace(name))
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.Get(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// DeleteByGVK 删除相关的业务逻辑。
func (s *DynamicResourceService) DeleteByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, name string) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	q := k.WithContext(ctx).Resource(u).Name(strings.TrimSpace(name))
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	return q.Delete().Error
}

// ResolveCRKindFromCRD 执行对应的业务逻辑。
func (s *DynamicResourceService) ResolveCRKindFromCRD(ctx context.Context, k *kom.Kubectl, group, version, resource string) (string, error) {
	group = strings.TrimSpace(group)
	version = strings.TrimSpace(version)
	resource = strings.TrimSpace(resource)
	if group == "" || version == "" || resource == "" {
		return "", apperror.BadRequest("group/version/resource 不能为空")
	}
	list, err := s.ListByGVK(ctx, k, schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}, "")
	if err != nil {
		return "", apperror.Internal(fmt.Sprintf("解析 CR 类型失败: %v", err))
	}
	for _, item := range list {
		var crd apiextv1.CustomResourceDefinition
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &crd); e != nil {
			continue
		}
		if strings.TrimSpace(crd.Spec.Group) != group {
			continue
		}
		if strings.TrimSpace(crd.Spec.Names.Plural) != resource {
			continue
		}
		for _, v := range crd.Spec.Versions {
			if strings.TrimSpace(v.Name) == version && v.Served {
				return strings.TrimSpace(crd.Spec.Names.Kind), nil
			}
		}
	}
	return "", apperror.BadRequest("未找到匹配的 CR 资源类型")
}

// ListCR 查询列表相关的业务逻辑。
func (s *DynamicResourceService) ListCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace string) ([]unstructured.Unstructured, error) {
	kind, err := s.ResolveCRKindFromCRD(ctx, k, group, version, resource)
	if err != nil {
		return nil, err
	}
	var list []unstructured.Unstructured
	q := k.WithContext(ctx).CRD(group, version, kind)
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetCR 获取相关的业务逻辑。
func (s *DynamicResourceService) GetCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {
	kind, err := s.ResolveCRKindFromCRD(ctx, k, group, version, resource)
	if err != nil {
		return nil, err
	}
	var obj unstructured.Unstructured
	q := k.WithContext(ctx).CRD(group, version, kind).Name(strings.TrimSpace(name))
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.Get(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// DeleteCR 删除相关的业务逻辑。
func (s *DynamicResourceService) DeleteCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace, name string) error {
	kind, err := s.ResolveCRKindFromCRD(ctx, k, group, version, resource)
	if err != nil {
		return err
	}
	q := k.WithContext(ctx).CRD(group, version, kind).Name(strings.TrimSpace(name))
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	}
	return q.Delete().Error
}

// ApplyManifest 提交申请相关的业务逻辑。
func (s *DynamicResourceService) ApplyManifest(ctx context.Context, k *kom.Kubectl, manifest string, exists func(context.Context) bool) error {
	if err := k.WithContext(ctx).Applier().Apply(manifest); err != nil {
		if k8sutil.IsLikelySuccessfulApplyError(err) {
			return nil
		}
		if exists != nil && exists(ctx) {
			return nil
		}
		return fmt.Errorf("%v", err)
	}
	return nil
}

// GVKByKind 执行对应的业务逻辑。
func (s *DynamicResourceService) GVKByKind(kind string) (schema.GroupVersionKind, bool) {
	switch strings.TrimSpace(kind) {
	case "Namespace":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, true
	case "ConfigMap":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}, true
	case "Secret":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}, true
	case "Ingress":
		return schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}, true
	case "Deployment":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, true
	case "StatefulSet":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}, true
	case "DaemonSet":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}, true
	case "Job":
		return schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}, true
	case "CronJob":
		return schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"}, true
	case "CustomResourceDefinition":
		return schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}, true
	default:
		return schema.GroupVersionKind{}, false
	}
}

// ExistsByKind 执行对应的业务逻辑。
func (s *DynamicResourceService) ExistsByKind(ctx context.Context, k *kom.Kubectl, kind, namespace, name string) bool {
	gvk, ok := s.GVKByKind(kind)
	if !ok || strings.TrimSpace(name) == "" {
		return false
	}
	_, err := s.GetByGVK(ctx, k, gvk, strings.TrimSpace(namespace), strings.TrimSpace(name))
	return err == nil || apierrors.IsAlreadyExists(err)
}
