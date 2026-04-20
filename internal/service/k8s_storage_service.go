package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/apperror"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type StorageListQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Keyword   string `form:"keyword"`
}

type StorageDetailQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Name      string `form:"name" binding:"required"`
}

type StorageApplyRequest = ClusterManifestApplyRequest
type StorageDeleteRequest struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Name      string `form:"name" binding:"required"`
}

type PersistentVolumeItem struct {
	Name          string `json:"name"`
	Capacity      string `json:"capacity"`
	AccessModes   string `json:"access_modes"`
	ReclaimPolicy string `json:"reclaim_policy"`
	Status        string `json:"status"`
	Claim         string `json:"claim,omitempty"`
	StorageClass  string `json:"storage_class,omitempty"`
	CreationTime  string `json:"creation_time"`
}

type PersistentVolumeClaimItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Volume       string `json:"volume,omitempty"`
	Capacity     string `json:"capacity,omitempty"`
	AccessModes  string `json:"access_modes,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
	CreationTime string `json:"creation_time"`
}

type StorageClassItem struct {
	Name                 string `json:"name"`
	Provisioner          string `json:"provisioner"`
	ReclaimPolicy        string `json:"reclaim_policy,omitempty"`
	VolumeBindingMode    string `json:"volume_binding_mode,omitempty"`
	AllowVolumeExpansion bool   `json:"allow_volume_expansion"`
	CreationTime         string `json:"creation_time"`
}

type StorageDetail struct {
	YAML string `json:"yaml"`
}

type K8sStorageService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sStorageService 创建相关逻辑。
func NewK8sStorageService(runtime *K8sRuntimeService) *K8sStorageService {
	return &K8sStorageService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var (
	pvGVK  = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolume"}
	pvcGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}
	scGVK  = schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}
)

// ListPVs 查询列表相关的业务逻辑。
func (s *K8sStorageService) ListPVs(ctx context.Context, q StorageListQuery) ([]PersistentVolumeItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, pvGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 PV 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]PersistentVolumeItem, 0, len(listU))
	for _, u := range listU {
		var obj corev1.PersistentVolume
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		capacity := ""
		if qv, ok := obj.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = qv.String()
		}
		claim := ""
		if obj.Spec.ClaimRef != nil {
			claim = strings.TrimSpace(obj.Spec.ClaimRef.Namespace + "/" + obj.Spec.ClaimRef.Name)
		}
		out = append(out, PersistentVolumeItem{
			Name:          obj.Name,
			Capacity:      fallbackDash(capacity),
			AccessModes:   joinPVModes(obj.Spec.AccessModes),
			ReclaimPolicy: string(obj.Spec.PersistentVolumeReclaimPolicy),
			Status:        string(obj.Status.Phase),
			Claim:         fallbackDash(claim),
			StorageClass:  fallbackDash(obj.Spec.StorageClassName),
			CreationTime:  obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ListPVCs 查询列表相关的业务逻辑。
func (s *K8sStorageService) ListPVCs(ctx context.Context, q StorageListQuery) ([]PersistentVolumeClaimItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	if ns == "" {
		return nil, apperror.BadRequest("命名空间不能为空")
	}
	listU, err := s.dyn.ListByGVK(ctx, k, pvcGVK, ns)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 PVC 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]PersistentVolumeClaimItem, 0, len(listU))
	for _, u := range listU {
		var obj corev1.PersistentVolumeClaim
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		capacity := ""
		if qv, ok := obj.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = qv.String()
		}
		storageClass := ""
		if obj.Spec.StorageClassName != nil {
			storageClass = *obj.Spec.StorageClassName
		}
		out = append(out, PersistentVolumeClaimItem{
			Name:         obj.Name,
			Namespace:    obj.Namespace,
			Status:       string(obj.Status.Phase),
			Volume:       fallbackDash(obj.Spec.VolumeName),
			Capacity:     fallbackDash(capacity),
			AccessModes:  joinPVModes(obj.Status.AccessModes),
			StorageClass: fallbackDash(storageClass),
			CreationTime: obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ListStorageClasses 查询列表相关的业务逻辑。
func (s *K8sStorageService) ListStorageClasses(ctx context.Context, q StorageListQuery) ([]StorageClassItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, scGVK, "")
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 StorageClass 列表失败: %v", err))
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]StorageClassItem, 0, len(listU))
	for _, u := range listU {
		var obj storagev1.StorageClass
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(obj.Name), kw) {
			continue
		}
		rp := ""
		if obj.ReclaimPolicy != nil {
			rp = string(*obj.ReclaimPolicy)
		}
		vb := ""
		if obj.VolumeBindingMode != nil {
			vb = string(*obj.VolumeBindingMode)
		}
		out = append(out, StorageClassItem{
			Name:                 obj.Name,
			Provisioner:          obj.Provisioner,
			ReclaimPolicy:        fallbackDash(rp),
			VolumeBindingMode:    fallbackDash(vb),
			AllowVolumeExpansion: obj.AllowVolumeExpansion != nil && *obj.AllowVolumeExpansion,
			CreationTime:         obj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sStorageService) Detail(ctx context.Context, kind string, q StorageDetailQuery) (*StorageDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, ns, err := resolveStorageGVKAndNS(kind, strings.TrimSpace(q.Namespace))
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, gvk, ns, strings.TrimSpace(q.Name))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取详情失败: %v", err))
	}
	obj := u.DeepCopy()
	obj.SetManagedFields(nil)
	y, _ := yaml.Marshal(obj.Object)
	return &StorageDetail{YAML: string(y)}, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sStorageService) Apply(ctx context.Context, req StorageApplyRequest) error {
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
func (s *K8sStorageService) Delete(ctx context.Context, kind string, req StorageDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	gvk, ns, err := resolveStorageGVKAndNS(kind, strings.TrimSpace(req.Namespace))
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, ns, strings.TrimSpace(req.Name)); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除失败: %v", err))
	}
	return nil
}

func resolveStorageGVKAndNS(kind, namespace string) (schema.GroupVersionKind, string, error) {
	switch strings.TrimSpace(kind) {
	case "PersistentVolume":
		return pvGVK, "", nil
	case "PersistentVolumeClaim":
		if namespace == "" {
			return schema.GroupVersionKind{}, "", apperror.BadRequest("命名空间不能为空")
		}
		return pvcGVK, namespace, nil
	case "StorageClass":
		return scGVK, "", nil
	default:
		return schema.GroupVersionKind{}, "", apperror.BadRequest("不支持的 kind")
	}
}

func joinPVModes(modes []corev1.PersistentVolumeAccessMode) string {
	if len(modes) == 0 {
		return "-"
	}
	out := make([]string, 0, len(modes))
	for _, m := range modes {
		out = append(out, string(m))
	}
	return strings.Join(out, ",")
}

func fallbackDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}
