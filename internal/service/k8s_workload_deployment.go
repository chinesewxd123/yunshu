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

func (s *K8sWorkloadService) ListDeployments(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Deployment")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt78bb8313c519)
	}
	list := make([]appsv1.Deployment, 0, len(listU))
	for _, item := range listU {
		var d appsv1.Deployment
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &d); e != nil {
			continue
		}
		list = append(list, d)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	deployUsage := aggregateDeploymentPodUsage(ctx, k, s.dyn, q.Namespace)
	out := make([]WorkloadItem, 0, len(list))
	for _, d := range list {
		if kw != "" && !strings.Contains(strings.ToLower(d.Name), kw) {
			continue
		}
		ready := fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas)
		replicas := fmt.Sprintf("%d", d.Status.Replicas)
		available := fmt.Sprintf("%d", d.Status.AvailableReplicas)
		updated := fmt.Sprintf("%d", d.Status.UpdatedReplicas)
		readyPercent := 0
		if d.Status.Replicas > 0 {
			readyPercent = int((float64(d.Status.ReadyReplicas) / float64(d.Status.Replicas)) * 100)
		}
		scale := int64(d.Status.Replicas)
		if scale < 1 {
			scale = 1
		}
		cpuUse, memUse, cr, cl, mr, ml := workloadUsagePercents(deployUsage[d.Name], d.Spec.Template.Spec, scale)
		out = append(out, WorkloadItem{
			Name:           d.Name,
			Namespace:      d.Namespace,
			Ready:          ready,
			Replicas:       replicas,
			Available:      available,
			Updated:        updated,
			ReadyPercent:   readyPercent,
			ResourceText:   deploymentResourceSummary(d),
			ContainersText: deploymentContainersSummary(d),
			ConditionsText: deploymentConditionsSummary(d),
			CPUUsage:       cpuUse,
			MemUsage:       memUse,
			CPUPctRequest:  cr,
			CPUPctLimit:    cl,
			MemPctRequest:  mr,
			MemPctLimit:    ml,
			Labels:         d.Labels,
			Annotations:    d.Annotations,
			CreationTime:   d.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(d.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) DeploymentDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Deployment")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmta3018a66177e)
	}
	var obj appsv1.Deployment
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	// managedFields 等字段会生成难以解析的 YAML（例如包含 f:xxx 这类 key），前端需要可稳定解析的 spec/metadata
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "apps/v1"
	copyObj.Kind = "Deployment"
	copyObj.ManagedFields = nil
	copyObj.Status = appsv1.DeploymentStatus{}
	y, _ := yaml.Marshal(copyObj)
	return &WorkloadDetail{YAML: string(y), Object: copyObj}, nil
}

// DeploymentScale 执行对应的业务逻辑。
// DeploymentScale 水平扩缩（修改 replicas）。
// 对齐 Kubernetes 中可通过 HPA / scale 子资源调整副本的控制器：Deployment、StatefulSet、ReplicaSet、ReplicationController；
// 不包含 DaemonSet、Job、CronJob（后三者不按「副本数」做持续水平伸缩）。
func (s *K8sWorkloadService) DeploymentScale(ctx context.Context, req WorkloadScaleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if req.Replicas < 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsgba0d4ada9f12)
	}
	var obj appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		return k8sFail("k8s.workload", "api", err)
	}
	copyObj := obj.DeepCopy()
	copyObj.Spec.Replicas = &req.Replicas
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail("k8s.workload", "api", err)
	}
	return nil
}

// DeploymentRestart 执行对应的业务逻辑。
func (s *K8sWorkloadService) DeploymentRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.Deployment{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		if apierrors.IsForbidden(err) {
			return constants.ErrForbiddenWithMsg(constants.ErrMsg4a3ba8680915)
		}
		return k8sFail("k8s.workload", "api", err)
	}
	return nil
}

// DeploymentPatchContainerResources 垂直扩缩：修改 Deployment Pod 模板内指定容器的 requests/limits（对齐 VPA 修改模板资源的范畴）。
func (s *K8sWorkloadService) DeploymentPatchContainerResources(ctx context.Context, req WorkloadContainerResourcesRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		return k8sFail("k8s.workload", "api", err)
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
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail("k8s.workload", "api", err)
	}
	return nil
}
