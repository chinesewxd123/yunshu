package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sutil"
	"yunshu/internal/service/svcerr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

func (s *K8sWorkloadService) ListDaemonSets(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("DaemonSet")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt22f7c7b69366)
	}
	list := make([]appsv1.DaemonSet, 0, len(listU))
	for _, item := range listU {
		var ds appsv1.DaemonSet
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ds); e != nil {
			continue
		}
		list = append(list, ds)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	dsUsage := aggregateDaemonSetPodUsage(ctx, k, s.dyn, q.Namespace)
	out := make([]WorkloadItem, 0, len(list))
	for _, ds := range list {
		if kw != "" && !strings.Contains(strings.ToLower(ds.Name), kw) {
			continue
		}
		ready := fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
		readyPercent := 0
		if ds.Status.DesiredNumberScheduled > 0 {
			readyPercent = int((float64(ds.Status.NumberReady) / float64(ds.Status.DesiredNumberScheduled)) * 100)
		}
		scale := int64(ds.Status.CurrentNumberScheduled)
		if scale < 1 {
			scale = 1
		}
		cpuUse, memUse, cr, cl, mr, ml := workloadUsagePercents(dsUsage[ds.Name], ds.Spec.Template.Spec, scale)
		out = append(out, WorkloadItem{
			Name:           ds.Name,
			Namespace:      ds.Namespace,
			Ready:          ready,
			Replicas:       fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled),
			Available:      fmt.Sprintf("%d", ds.Status.NumberAvailable),
			Updated:        fmt.Sprintf("%d", ds.Status.UpdatedNumberScheduled),
			ReadyPercent:   readyPercent,
			ResourceText:   podTemplateResourceSummary(ds.Spec.Template.Spec),
			ContainersText: podTemplateContainersSummary(ds.Spec.Template.Spec),
			ConditionsText: daemonSetConditionsSummary(ds),
			CPUUsage:       cpuUse,
			MemUsage:       memUse,
			CPUPctRequest:  cr,
			CPUPctLimit:    cl,
			MemPctRequest:  mr,
			MemPctLimit:    ml,
			Labels:         ds.Labels,
			Annotations:    ds.Annotations,
			CreationTime:   ds.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(ds.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// DaemonSetDetail 执行对应的业务逻辑。
func (s *K8sWorkloadService) DaemonSetDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("DaemonSet")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg728030d27854)
		}
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt960ced5a2f6f)
	}
	var obj appsv1.DaemonSet
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "apps/v1"
	copyObj.Kind = "DaemonSet"
	copyObj.ManagedFields = nil
	copyObj.Status = appsv1.DaemonSetStatus{}
	y, _ := yaml.Marshal(copyObj)
	return &WorkloadDetail{YAML: string(y), Object: copyObj}, nil
}

// DaemonSetRestart 执行对应的业务逻辑。
func (s *K8sWorkloadService) DaemonSetRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.DaemonSet{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg728030d27854)
		}
		if apierrors.IsForbidden(err) {
			return constants.ErrForbiddenWithMsg(constants.ErrMsg6e28e4e09c23)
		}
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// ListJobs 查询列表相关的业务逻辑。
func (s *K8sWorkloadService) DaemonSetPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var ds appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(q.Namespace).Name(q.Name).Get(&ds).Error; err != nil {
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt960ced5a2f6f)
	}
	selector := metav1.FormatLabelSelector(ds.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

// JobPods 执行对应的业务逻辑。
func (s *K8sWorkloadService) DaemonSetPatchContainerResources(ctx context.Context, req WorkloadContainerResourcesRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg728030d27854)
		}
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	containers := obj.Spec.Template.Spec.Containers
	idx := workloadContainerIndex(containers, req.ContainerName)
	if idx < 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg1a5aaa6cfa35)
	}
	copyObj := obj.DeepCopy()
	c := &copyObj.Spec.Template.Spec.Containers[idx]
	if c.Resources.Requests == nil {
		c.Resources.Requests = corev1.ResourceList{}
	}
	if c.Resources.Limits == nil {
		c.Resources.Limits = corev1.ResourceList{}
	}
	if err := k8sutil.PatchResourceList(&c.Resources.Requests, req.Requests); err != nil {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte922f3829384, err))
	}
	if err := k8sutil.PatchResourceList(&c.Resources.Limits, req.Limits); err != nil {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmt81f1534a632d, err))
	}
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// JobPatchContainerResources 垂直扩缩：修改 Job Pod 模板内指定容器的 requests/limits。
// 批量类工作负载若由集群 VPA 纳管，通常仅在 Initial/Off 等模式下对新建 Pod 生效更安全；此处仅修改模板，不调整并行语义。
