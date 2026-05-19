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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

func (s *K8sWorkloadService) ListStatefulSets(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("StatefulSet")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt3bef5bb60df3)
	}
	list := make([]appsv1.StatefulSet, 0, len(listU))
	for _, item := range listU {
		var st appsv1.StatefulSet
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &st); e != nil {
			continue
		}
		list = append(list, st)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	stsUsage := aggregateStatefulSetPodUsage(ctx, k, s.dyn, q.Namespace)
	out := make([]WorkloadItem, 0, len(list))
	for _, st := range list {
		if kw != "" && !strings.Contains(strings.ToLower(st.Name), kw) {
			continue
		}
		ready := fmt.Sprintf("%d/%d", st.Status.ReadyReplicas, st.Status.Replicas)
		replicas := fmt.Sprintf("%d", st.Status.Replicas)
		readyPercent := 0
		if st.Status.Replicas > 0 {
			readyPercent = int((float64(st.Status.ReadyReplicas) / float64(st.Status.Replicas)) * 100)
		}
		scale := int64(st.Status.Replicas)
		if scale < 1 {
			scale = 1
		}
		cpuUse, memUse, cr, cl, mr, ml := workloadUsagePercents(stsUsage[st.Name], st.Spec.Template.Spec, scale)
		out = append(out, WorkloadItem{
			Name:           st.Name,
			Namespace:      st.Namespace,
			Ready:          ready,
			Replicas:       replicas,
			Available:      fmt.Sprintf("%d", st.Status.AvailableReplicas),
			Updated:        fmt.Sprintf("%d", st.Status.UpdatedReplicas),
			ReadyPercent:   readyPercent,
			ResourceText:   podTemplateResourceSummary(st.Spec.Template.Spec),
			ContainersText: podTemplateContainersSummary(st.Spec.Template.Spec),
			ConditionsText: statefulSetConditionsSummary(st),
			CPUUsage:       cpuUse,
			MemUsage:       memUse,
			CPUPctRequest:  cr,
			CPUPctLimit:    cl,
			MemPctRequest:  mr,
			MemPctLimit:    ml,
			Labels:         st.Labels,
			Annotations:    st.Annotations,
			CreationTime:   st.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(st.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// StatefulSetDetail 执行对应的业务逻辑。
func (s *K8sWorkloadService) StatefulSetDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("StatefulSet")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg728d3e3b08a7)
		}
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt70dba6fa52bd)
	}
	var obj appsv1.StatefulSet
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "apps/v1"
	copyObj.Kind = "StatefulSet"
	copyObj.ManagedFields = nil
	copyObj.Status = appsv1.StatefulSetStatus{}
	y, _ := yaml.Marshal(copyObj)
	return &WorkloadDetail{YAML: string(y), Object: copyObj}, nil
}

// StatefulSetScale 执行对应的业务逻辑。
// StatefulSetScale 水平扩缩（修改 replicas）。语义同 DeploymentScale，属 HPA scale 子资源一类。
func (s *K8sWorkloadService) StatefulSetScale(ctx context.Context, req WorkloadScaleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if req.Replicas < 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsgba0d4ada9f12)
	}
	var obj appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg728d3e3b08a7)
		}
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	copyObj := obj.DeepCopy()
	copyObj.Spec.Replicas = &req.Replicas
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// StatefulSetPatchContainerResources 垂直扩缩：修改 StatefulSet Pod 模板内指定容器的 requests/limits（对齐 VPA 范畴）。
func (s *K8sWorkloadService) StatefulSetPatchContainerResources(ctx context.Context, req WorkloadContainerResourcesRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg728d3e3b08a7)
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
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// StatefulSetRestart 执行对应的业务逻辑。
func (s *K8sWorkloadService) StatefulSetRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.StatefulSet{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg728d3e3b08a7)
		}
		if apierrors.IsForbidden(err) {
			return constants.ErrForbiddenWithMsg(constants.ErrMsga0421725a51e)
		}
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// ListDaemonSets 查询列表相关的业务逻辑。
