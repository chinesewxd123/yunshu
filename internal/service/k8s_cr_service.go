package service

import (
	"context"
	"sort"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

type CRResourceListQuery = ClusterOnlyQuery

type CRResourceBaseQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Group     string `form:"group" binding:"required"`
	Version   string `form:"version" binding:"required"`
	Resource  string `form:"resource" binding:"required"`
	Namespace string `form:"namespace"`
}

type CRListQuery struct {
	CRResourceBaseQuery
	Keyword string `form:"keyword"`
}

type CRDetailQuery struct {
	CRResourceBaseQuery
	Name string `form:"name" binding:"required"`
}

type CRDeleteRequest = CRDetailQuery

type CRApplyRequest = ClusterManifestApplyRequest

type CRResourceItem struct {
	Name       string `json:"name"`
	Group      string `json:"group"`
	Version    string `json:"version"`
	Resource   string `json:"resource"`
	Kind       string `json:"kind"`
	Scope      string `json:"scope"`
	Namespaced bool   `json:"namespaced"`
}

type CRItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace,omitempty"`
	APIVersion   string `json:"api_version"`
	Kind         string `json:"kind"`
	CreationTime string `json:"creation_time"`
}

type CRDetail struct {
	Item CRItem `json:"item"`
	YAML string `json:"yaml"`
}

type K8sCRService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sCRService 创建相关逻辑。
func NewK8sCRService(runtime *K8sRuntimeService) *K8sCRService {
	return &K8sCRService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

// ListResources 查询列表相关的业务逻辑。
func (s *K8sCRService) ListResources(ctx context.Context, q CRResourceListQuery) ([]CRResourceItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, crdGVK, "")
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.cr", "api", err, constants.ErrFmtcf9172f6e822)
	}
	list := make([]apiextv1.CustomResourceDefinition, 0, len(listU))
	for _, item := range listU {
		var crd apiextv1.CustomResourceDefinition
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &crd); e != nil {
			continue
		}
		list = append(list, crd)
	}
	out := make([]CRResourceItem, 0, len(list))
	for _, crd := range list {
		if !isCRDEstablished(&crd) {
			continue
		}
		version := ""
		for _, v := range crd.Spec.Versions {
			if v.Storage {
				version = v.Name
				break
			}
		}
		if version == "" && len(crd.Spec.Versions) > 0 {
			version = crd.Spec.Versions[0].Name
		}
		if version == "" {
			continue
		}
		namespaced := strings.EqualFold(string(crd.Spec.Scope), "Namespaced")
		out = append(out, CRResourceItem{
			Name:       crd.Name,
			Group:      crd.Spec.Group,
			Version:    version,
			Resource:   crd.Spec.Names.Plural,
			Kind:       crd.Spec.Names.Kind,
			Scope:      string(crd.Spec.Scope),
			Namespaced: namespaced,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// List 查询列表相关的业务逻辑。
func (s *K8sCRService) List(ctx context.Context, q CRListQuery) ([]CRItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	list, err := s.dyn.ListCR(ctx, k, q.Group, q.Version, q.Resource, q.Namespace)
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.cr", "api", err, constants.ErrFmt0f0376756412)
	}

	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]CRItem, 0, len(list))
	for _, item := range list {
		name := item.GetName()
		if kw != "" && !strings.Contains(strings.ToLower(name), kw) {
			continue
		}
		out = append(out, CRItem{
			Name:         name,
			Namespace:    item.GetNamespace(),
			APIVersion:   item.GetAPIVersion(),
			Kind:         item.GetKind(),
			CreationTime: item.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sCRService) Detail(ctx context.Context, q CRDetailQuery) (*CRDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	obj, err := s.dyn.GetCR(ctx, k, q.Group, q.Version, q.Resource, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg0646b6d79607)
		}
		return nil, svcerr.Internal(ctx, "k8s.cr", "api", err, constants.ErrFmt076435509f8c)
	}
	copyObj := obj.DeepCopy()
	unstructured.RemoveNestedField(copyObj.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(copyObj.Object, "status")
	y, _ := yaml.Marshal(copyObj.Object)
	return &CRDetail{
		Item: CRItem{
			Name:         copyObj.GetName(),
			Namespace:    copyObj.GetNamespace(),
			APIVersion:   copyObj.GetAPIVersion(),
			Kind:         copyObj.GetKind(),
			CreationTime: copyObj.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05"),
		},
		YAML: string(y),
	}, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sCRService) Apply(ctx context.Context, req CRApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, nil); err != nil {
		return k8sFail(ctx, "k8s.cr", "api", err)
	}
	return nil
}

// Delete 删除相关的业务逻辑。
func (s *K8sCRService) Delete(ctx context.Context, req CRDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	err = s.dyn.DeleteCR(ctx, k, req.Group, req.Version, req.Resource, req.Namespace, req.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.cr", "api", err)
	}
	return nil
}
