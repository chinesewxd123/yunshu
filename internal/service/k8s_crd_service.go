package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type CRDListQuery = ClusterKeywordQuery
type CRDDetailQuery = ClusterNameQuery
type CRDApplyRequest = ClusterManifestApplyRequest
type CRDDeleteRequest = ClusterNameQuery

type CRDItem struct {
	Name           string `json:"name"`
	Group          string `json:"group"`
	Scope          string `json:"scope"`
	Kind           string `json:"kind"`
	Plural         string `json:"plural"`
	CurrentVersion string `json:"current_version"`
	Established    bool   `json:"established"`
	CreationTime   string `json:"creation_time"`
}

type CRDDetail struct {
	Item CRDItem `json:"item"`
	YAML string  `json:"yaml"`
}

type K8sCRDService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sCRDService 创建相关逻辑。
func NewK8sCRDService(runtime *K8sRuntimeService) *K8sCRDService {
	return &K8sCRDService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var crdGVK = schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}

// List 查询列表相关的业务逻辑。
func (s *K8sCRDService) List(ctx context.Context, q CRDListQuery) ([]CRDItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ul, err := s.dyn.ListByGVK(ctx, k, crdGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 CRD 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]CRDItem, 0, len(ul))
	for _, item := range ul {
		var crd apiextv1.CustomResourceDefinition
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &crd); e != nil {
			continue
		}
		if kw != "" &&
			!strings.Contains(strings.ToLower(crd.Name), kw) &&
			!strings.Contains(strings.ToLower(crd.Spec.Group), kw) &&
			!strings.Contains(strings.ToLower(crd.Spec.Names.Kind), kw) {
			continue
		}
		out = append(out, crdToItem(&crd))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sCRDService) Detail(ctx context.Context, q CRDDetailQuery) (*CRDDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, crdGVK, "", q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("CRD 资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 CRD 详情失败: %v", err))
	}
	var obj apiextv1.CustomResourceDefinition
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "apiextensions.k8s.io/v1"
	copyObj.Kind = "CustomResourceDefinition"
	copyObj.ManagedFields = nil
	copyObj.Status = apiextv1.CustomResourceDefinitionStatus{}
	y, _ := yaml.Marshal(copyObj)
	return &CRDDetail{
		Item: crdToItem(copyObj),
		YAML: string(y),
	}, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sCRDService) Apply(ctx context.Context, req CRDApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("资源清单不能为空")
	}
	refs := extractCRDRefs(req.Manifest)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, name := range refs {
			_, e := s.dyn.GetByGVK(c, k, crdGVK, "", name)
			if e != nil {
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

// Delete 删除相关的业务逻辑。
func (s *K8sCRDService) Delete(ctx context.Context, req CRDDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, crdGVK, "", req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 CRD 失败: %v", err))
	}
	return nil
}

func crdToItem(crd *apiextv1.CustomResourceDefinition) CRDItem {
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
	return CRDItem{
		Name:           crd.Name,
		Group:          crd.Spec.Group,
		Scope:          string(crd.Spec.Scope),
		Kind:           crd.Spec.Names.Kind,
		Plural:         crd.Spec.Names.Plural,
		CurrentVersion: version,
		Established:    isCRDEstablished(crd),
		CreationTime:   crd.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}

func isCRDEstablished(crd *apiextv1.CustomResourceDefinition) bool {
	for _, c := range crd.Status.Conditions {
		if c.Type == apiextv1.Established && c.Status == apiextv1.ConditionTrue {
			return true
		}
	}
	return false
}

func extractCRDRefs(manifest string) []string {
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
		if strings.TrimSpace(kind) != "CustomResourceDefinition" {
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
