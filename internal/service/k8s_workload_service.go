package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

type NamespacedListQuery struct {
	ClusterNamespaceKeywordQuery
	LabelQuery string `form:"label_selector"`
}

type NamespacedDetailQuery = ClusterNamespaceNameQuery
type NamespacedApplyRequest = ClusterManifestApplyRequest
type NamespacedDeleteRequest = ClusterNamespaceNameQuery

type WorkloadScaleRequest = ClusterNamespaceNameScaleRequest
type CronJobSuspendRequest = ClusterNamespaceNameSuspendRequest

type CronJobTriggerRequest = ClusterNamespaceNameRequest
type JobRerunRequest = ClusterNamespaceNameRequest

type WorkloadItem struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Ready          string            `json:"ready,omitempty"`
	Replicas       string            `json:"replicas,omitempty"`
	Available      string            `json:"available,omitempty"`
	Updated        string            `json:"updated,omitempty"`
	ReadyPercent   int               `json:"ready_percent,omitempty"`
	ResourceText   string            `json:"resource_text,omitempty"`
	ContainersText string            `json:"containers_text,omitempty"`
	ConditionsText string            `json:"conditions_text,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Active         string            `json:"active,omitempty"`
	Failed         string            `json:"failed,omitempty"`
	StartTime      string            `json:"start_time,omitempty"`
	CompletionTime string            `json:"completion_time,omitempty"`
	Age            string            `json:"age,omitempty"`
	CreationTime   string            `json:"creation_time"`
}

type CronJobItem struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Schedule           string            `json:"schedule"`
	Suspend            bool              `json:"suspend"`
	ReadyPercent       int               `json:"ready_percent,omitempty"`
	ResourceText       string            `json:"resource_text,omitempty"`
	ContainersText     string            `json:"containers_text,omitempty"`
	ConditionsText     string            `json:"conditions_text,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	LastScheduleTime   string            `json:"last_schedule_time,omitempty"`
	LastSuccessfulTime string            `json:"last_successful_time,omitempty"`
	ActiveCount        string            `json:"active_count,omitempty"`
	Age                string            `json:"age,omitempty"`
	CreationTime       string            `json:"creation_time"`
}

type WorkloadDetail struct {
	YAML   string `json:"yaml"`
	Object any    `json:"object,omitempty"`
}

type K8sWorkloadService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sWorkloadService(runtime *K8sRuntimeService) *K8sWorkloadService {
	return &K8sWorkloadService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

func (s *K8sWorkloadService) ListDeployments(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Deployment")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Deployments 失败: %v", err))
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
			Labels:         d.Labels,
			Annotations:    d.Annotations,
			CreationTime:   d.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(d.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func deploymentResourceSummary(d appsv1.Deployment) string {
	var reqCPU resource.Quantity
	var reqMem resource.Quantity
	var limCPU resource.Quantity
	var limMem resource.Quantity
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Resources.Requests != nil {
			reqCPU.Add(c.Resources.Requests[corev1.ResourceCPU])
			reqMem.Add(c.Resources.Requests[corev1.ResourceMemory])
		}
		if c.Resources.Limits != nil {
			limCPU.Add(c.Resources.Limits[corev1.ResourceCPU])
			limMem.Add(c.Resources.Limits[corev1.ResourceMemory])
		}
	}
	cpuReq := "-"
	cpuLim := "-"
	memReq := "-"
	memLim := "-"
	if !reqCPU.IsZero() {
		cpuReq = reqCPU.String()
	}
	if !limCPU.IsZero() {
		cpuLim = limCPU.String()
	}
	if !reqMem.IsZero() {
		memReq = reqMem.String()
	}
	if !limMem.IsZero() {
		memLim = limMem.String()
	}
	return fmt.Sprintf("CPU: %s / %s\n内存: %s / %s", cpuReq, cpuLim, memReq, memLim)
}

func deploymentContainersSummary(d appsv1.Deployment) string {
	if len(d.Spec.Template.Spec.Containers) == 0 {
		return "-"
	}
	out := make([]string, 0, len(d.Spec.Template.Spec.Containers))
	for _, c := range d.Spec.Template.Spec.Containers {
		image := strings.TrimSpace(c.Image)
		if image == "" {
			image = "-"
		}
		out = append(out, fmt.Sprintf("%s: %s", c.Name, image))
	}
	return strings.Join(out, "\n")
}

func deploymentConditionsSummary(d appsv1.Deployment) string {
	if len(d.Status.Conditions) == 0 {
		return "-"
	}
	out := make([]string, 0, len(d.Status.Conditions))
	for _, c := range d.Status.Conditions {
		out = append(out, fmt.Sprintf("%s=%s", c.Type, c.Status))
	}
	return strings.Join(out, ", ")
}

func podTemplateResourceSummary(spec corev1.PodSpec) string {
	cpuReq := *resource.NewQuantity(0, resource.DecimalSI)
	cpuLim := *resource.NewQuantity(0, resource.DecimalSI)
	memReq := *resource.NewQuantity(0, resource.BinarySI)
	memLim := *resource.NewQuantity(0, resource.BinarySI)
	for _, c := range spec.Containers {
		if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
			cpuReq.Add(q)
		}
		if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
			cpuLim.Add(q)
		}
		if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
			memReq.Add(q)
		}
		if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
			memLim.Add(q)
		}
	}
	return fmt.Sprintf("CPU: %s / %s\n内存: %s / %s", quantityOrDash(cpuReq), quantityOrDash(cpuLim), quantityOrDash(memReq), quantityOrDash(memLim))
}

func podTemplateContainersSummary(spec corev1.PodSpec) string {
	if len(spec.Containers) == 0 {
		return "-"
	}
	out := make([]string, 0, len(spec.Containers))
	for _, c := range spec.Containers {
		image := strings.TrimSpace(c.Image)
		if image == "" {
			image = "-"
		}
		out = append(out, fmt.Sprintf("%s: %s", c.Name, image))
	}
	return strings.Join(out, "\n")
}

func statefulSetConditionsSummary(st appsv1.StatefulSet) string {
	if len(st.Status.Conditions) == 0 {
		return "-"
	}
	out := make([]string, 0, len(st.Status.Conditions))
	for _, c := range st.Status.Conditions {
		out = append(out, fmt.Sprintf("%s=%s", c.Type, c.Status))
	}
	return strings.Join(out, ", ")
}

func daemonSetConditionsSummary(ds appsv1.DaemonSet) string {
	if len(ds.Status.Conditions) == 0 {
		return "-"
	}
	out := make([]string, 0, len(ds.Status.Conditions))
	for _, c := range ds.Status.Conditions {
		out = append(out, fmt.Sprintf("%s=%s", c.Type, c.Status))
	}
	return strings.Join(out, ", ")
}

func jobConditionsSummary(j batchv1.Job) string {
	if len(j.Status.Conditions) == 0 {
		return "-"
	}
	out := make([]string, 0, len(j.Status.Conditions))
	for _, c := range j.Status.Conditions {
		out = append(out, fmt.Sprintf("%s=%s", c.Type, c.Status))
	}
	return strings.Join(out, ", ")
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
			return nil, apperror.BadRequest("Deployment 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Deployment 失败: %v", err))
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

func (s *K8sWorkloadService) DeploymentScale(ctx context.Context, req WorkloadScaleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if req.Replicas < 0 {
		return apperror.BadRequest("replicas 不能小于 0")
	}
	var obj appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("Deployment 不存在")
		}
		return apperror.Internal(fmt.Sprintf("获取 Deployment 失败: %v", err))
	}
	copyObj := obj.DeepCopy()
	copyObj.Spec.Replicas = &req.Replicas
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("Deployment 扩缩容失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) DeploymentRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.Deployment{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("Deployment 不存在")
		}
		if apierrors.IsForbidden(err) {
			return apperror.Forbidden("无权限重启该 Deployment")
		}
		return apperror.Internal(fmt.Sprintf("Deployment 重启失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) ListStatefulSets(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("StatefulSet")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 StatefulSets 失败: %v", err))
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
			Labels:         st.Labels,
			Annotations:    st.Annotations,
			CreationTime:   st.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(st.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) StatefulSetDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("StatefulSet")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("StatefulSet 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 StatefulSet 失败: %v", err))
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

func (s *K8sWorkloadService) StatefulSetScale(ctx context.Context, req WorkloadScaleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if req.Replicas < 0 {
		return apperror.BadRequest("replicas 不能小于 0")
	}
	var obj appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("StatefulSet 不存在")
		}
		return apperror.Internal(fmt.Sprintf("获取 StatefulSet 失败: %v", err))
	}
	copyObj := obj.DeepCopy()
	copyObj.Spec.Replicas = &req.Replicas
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("StatefulSet 扩缩容失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) StatefulSetRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.StatefulSet{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("StatefulSet 不存在")
		}
		if apierrors.IsForbidden(err) {
			return apperror.Forbidden("无权限重启该 StatefulSet")
		}
		return apperror.Internal(fmt.Sprintf("StatefulSet 重启失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) ListDaemonSets(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("DaemonSet")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 DaemonSets 失败: %v", err))
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
			Labels:         ds.Labels,
			Annotations:    ds.Annotations,
			CreationTime:   ds.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(ds.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) DaemonSetDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("DaemonSet")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("DaemonSet 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 DaemonSet 失败: %v", err))
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

func (s *K8sWorkloadService) DaemonSetRestart(ctx context.Context, q NamespacedDetailQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(q.Namespace).Name(q.Name).
		Patch(&appsv1.DaemonSet{}, types.StrategicMergePatchType, patch).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("DaemonSet 不存在")
		}
		if apierrors.IsForbidden(err) {
			return apperror.Forbidden("无权限重启该 DaemonSet")
		}
		return apperror.Internal(fmt.Sprintf("DaemonSet 重启失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) ListJobs(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Job")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Jobs 失败: %v", err))
	}
	list := make([]batchv1.Job, 0, len(listU))
	for _, item := range listU {
		var j batchv1.Job
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &j); e != nil {
			continue
		}
		list = append(list, j)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]WorkloadItem, 0, len(list))
	for _, j := range list {
		if kw != "" && !strings.Contains(strings.ToLower(j.Name), kw) {
			continue
		}
		replicas := fmt.Sprintf("%d", k8sutil.Deref32(j.Spec.Completions))
		ready := fmt.Sprintf("%d/%d", j.Status.Succeeded, k8sutil.Deref32(j.Spec.Completions))

		active := fmt.Sprintf("%d", j.Status.Active)
		failed := fmt.Sprintf("%d", j.Status.Failed)

		start := ""
		if j.Status.StartTime != nil && !j.Status.StartTime.IsZero() {
			start = j.Status.StartTime.Time.Format("2006-01-02 15:04:05")
		}
		completion := ""
		if j.Status.CompletionTime != nil && !j.Status.CompletionTime.IsZero() {
			completion = j.Status.CompletionTime.Time.Format("2006-01-02 15:04:05")
		}

		out = append(out, WorkloadItem{
			Name:      j.Name,
			Namespace: j.Namespace,
			Ready:     ready,
			Replicas:  replicas,
			Available: fmt.Sprintf("%d", j.Status.Succeeded),
			Updated:   fmt.Sprintf("%d", j.Status.Active),
			ReadyPercent: func() int {
				total := k8sutil.Deref32(j.Spec.Completions)
				if total <= 0 {
					return 0
				}
				return int((float64(j.Status.Succeeded) / float64(total)) * 100)
			}(),
			ResourceText:   podTemplateResourceSummary(j.Spec.Template.Spec),
			ContainersText: podTemplateContainersSummary(j.Spec.Template.Spec),
			ConditionsText: jobConditionsSummary(j),
			Labels:         j.Labels,
			Annotations:    j.Annotations,
			Active:         active,
			Failed:         failed,
			StartTime:      start,
			CompletionTime: completion,
			CreationTime:   j.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:            k8sutil.HumanAge(j.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) JobDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Job")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("Job 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Job 失败: %v", err))
	}
	var obj batchv1.Job
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "batch/v1"
	copyObj.Kind = "Job"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &WorkloadDetail{YAML: string(y), Object: copyObj}, nil
}

func (s *K8sWorkloadService) ListCronJobs(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 CronJobs 失败: %v", err))
	}
	list := make([]batchv1.CronJob, 0, len(listU))
	for _, item := range listU {
		var cj batchv1.CronJob
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cj); e != nil {
			continue
		}
		list = append(list, cj)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	// keep compatibility with old return type by mapping to WorkloadItem; we will also add a typed endpoint later if needed
	out := make([]WorkloadItem, 0, len(list))
	for _, cj := range list {
		if kw != "" && !strings.Contains(strings.ToLower(cj.Name), kw) {
			continue
		}
		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}
		last := ""
		if cj.Status.LastScheduleTime != nil && !cj.Status.LastScheduleTime.IsZero() {
			last = cj.Status.LastScheduleTime.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, WorkloadItem{
			Name:      cj.Name,
			Namespace: cj.Namespace,
			Replicas: fmt.Sprintf("%s%s", cj.Spec.Schedule, func() string {
				if suspend {
					return "（暂停）"
				}
				return ""
			}()),
			CreationTime: cj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:          k8sutil.HumanAge(cj.CreationTimestamp.Time),
		})
		_ = last // keep for backward compatible list; v2 exposes it
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) ListCronJobsV2(ctx context.Context, q NamespacedListQuery) ([]CronJobItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 CronJobs 失败: %v", err))
	}
	list := make([]batchv1.CronJob, 0, len(listU))
	for _, item := range listU {
		var cj batchv1.CronJob
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cj); e != nil {
			continue
		}
		list = append(list, cj)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]CronJobItem, 0, len(list))
	for _, cj := range list {
		if kw != "" && !strings.Contains(strings.ToLower(cj.Name), kw) {
			continue
		}
		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}
		last := ""
		if cj.Status.LastScheduleTime != nil && !cj.Status.LastScheduleTime.IsZero() {
			last = cj.Status.LastScheduleTime.Time.Format("2006-01-02 15:04:05")
		}

		lastSuccess := ""
		if cj.Status.LastSuccessfulTime != nil && !cj.Status.LastSuccessfulTime.IsZero() {
			lastSuccess = cj.Status.LastSuccessfulTime.Time.Format("2006-01-02 15:04:05")
		}

		activeCount := fmt.Sprintf("%d", len(cj.Status.Active))

		out = append(out, CronJobItem{
			Name:               cj.Name,
			Namespace:          cj.Namespace,
			Schedule:           cj.Spec.Schedule,
			Suspend:            suspend,
			ReadyPercent:       100,
			ResourceText:       podTemplateResourceSummary(cj.Spec.JobTemplate.Spec.Template.Spec),
			ContainersText:     podTemplateContainersSummary(cj.Spec.JobTemplate.Spec.Template.Spec),
			ConditionsText:     "-",
			Labels:             cj.Labels,
			Annotations:        cj.Annotations,
			LastScheduleTime:   last,
			LastSuccessfulTime: lastSuccess,
			ActiveCount:        activeCount,
			CreationTime:       cj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:                k8sutil.HumanAge(cj.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) JobRerun(ctx context.Context, req JobRerunRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	var job batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Name(req.Name).Get(&job).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return "", apperror.BadRequest("Job 不存在")
		}
		return "", apperror.Internal(fmt.Sprintf("获取 Job 失败: %v", err))
	}
	newName := fmt.Sprintf("%s-rerun-%d", job.Name, time.Now().Unix())
	newJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newName,
			Namespace: req.Namespace,
			Labels:    job.Labels,
		},
		Spec: job.Spec,
	}
	// 清理不可拷贝字段/选择器冲突风险
	newJob.Spec.Selector = nil
	newJob.Spec.ManualSelector = nil
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Create(newJob).Error; err != nil {
		return "", apperror.Internal(fmt.Sprintf("重新执行 Job 失败: %v", err))
	}
	return newName, nil
}

type RelatedPodsQuery = ClusterNamespaceNameQuery

type RelatedPodItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Phase        string `json:"phase"`
	NodeName     string `json:"node_name"`
	PodIP        string `json:"pod_ip"`
	RestartCount int32  `json:"restart_count"`
	StartTime    string `json:"start_time,omitempty"`
}

func (s *K8sWorkloadService) DeploymentPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var d appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).Get(&d).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Deployment 失败: %v", err))
	}
	selector := metav1.FormatLabelSelector(d.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

func (s *K8sWorkloadService) StatefulSetPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var st appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(q.Namespace).Name(q.Name).Get(&st).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 StatefulSet 失败: %v", err))
	}
	selector := metav1.FormatLabelSelector(st.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

func (s *K8sWorkloadService) DaemonSetPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var ds appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(q.Namespace).Name(q.Name).Get(&ds).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 DaemonSet 失败: %v", err))
	}
	selector := metav1.FormatLabelSelector(ds.Spec.Selector)
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

func (s *K8sWorkloadService) JobPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var job batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(q.Namespace).Name(q.Name).Get(&job).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Job 失败: %v", err))
	}
	selector := ""
	if job.Spec.Selector != nil {
		selector = metav1.FormatLabelSelector(job.Spec.Selector)
	}
	if strings.TrimSpace(selector) == "" {
		selector = fmt.Sprintf("job-name=%s", job.Name)
	}
	return listPodsBySelector(ctx, k, q.Namespace, selector)
}

func (s *K8sWorkloadService) CronJobPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var cj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(q.Namespace).Name(q.Name).Get(&cj).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 CronJob 失败: %v", err))
	}

	// CronJob 触发后，Pod 通常由 Job 创建并带有 job-name 标签；
	// 直接按 cronjob 标签查 Pod 在部分版本/场景会为空，因此改为“先找 Job，再找 Pod”。
	var jobs []batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(q.Namespace).List(&jobs).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 CronJob 关联 Jobs 失败: %v", err))
	}

	all := make([]RelatedPodItem, 0)
	seen := map[string]bool{}
	for _, job := range jobs {
		belong := false
		for _, owner := range job.OwnerReferences {
			if owner.Kind == "CronJob" && owner.Name == cj.Name {
				belong = true
				break
			}
		}
		if !belong {
			continue
		}

		items, err := listPodsBySelector(ctx, k, q.Namespace, fmt.Sprintf("job-name=%s", job.Name))
		if err != nil {
			continue
		}
		for _, p := range items {
			key := p.Namespace + "/" + p.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, p)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all, nil
}

func listPodsBySelector(ctx context.Context, k *kom.Kubectl, namespace, selector string) ([]RelatedPodItem, error) {
	if k == nil {
		return nil, apperror.Internal("k8s 客户端不存在")
	}
	opts := metav1.ListOptions{}
	if strings.TrimSpace(selector) != "" {
		opts.LabelSelector = strings.TrimSpace(selector)
	}
	var list []corev1.Pod
	query := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(namespace)
	if strings.TrimSpace(opts.LabelSelector) != "" {
		query = query.WithLabelSelector(strings.TrimSpace(opts.LabelSelector))
	}
	if err := query.List(&list).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取关联 Pods 失败: %v", err))
	}
	out := make([]RelatedPodItem, 0, len(list))
	for _, p := range list {
		restarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		start := ""
		if p.Status.StartTime != nil && !p.Status.StartTime.IsZero() {
			start = p.Status.StartTime.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, RelatedPodItem{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Phase:        string(p.Status.Phase),
			NodeName:     p.Spec.NodeName,
			PodIP:        p.Status.PodIP,
			RestartCount: restarts,
			StartTime:    start,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sWorkloadService) CronJobDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("CronJob 不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 CronJob 失败: %v", err))
	}
	var obj batchv1.CronJob
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "batch/v1"
	copyObj.Kind = "CronJob"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &WorkloadDetail{YAML: string(y), Object: copyObj}, nil
}

func (s *K8sWorkloadService) Apply(ctx context.Context, req NamespacedApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("manifest 不能为空")
	}

	refs := extractWorkloadRefsForApply(req.Manifest)

	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, r := range refs {
			if strings.TrimSpace(r.Name) == "" {
				continue
			}
			if !s.dyn.ExistsByKind(c, k, r.Kind, r.Namespace, r.Name) {
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

type workloadRef struct {
	Kind      string
	Name      string
	Namespace string
}

func extractWorkloadRefsForApply(manifest string) []workloadRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]workloadRef, 0)

	seen := map[string]bool{}
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
		kind = strings.TrimSpace(kind)
		if kind == "" {
			continue
		}

		// 只兜底这些资源（与本接口对应）
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
		default:
			continue
		}

		metadata, _ := m["metadata"].(map[string]any)
		if metadata == nil {
			continue
		}

		name, _ := metadata["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		ns, _ := metadata["namespace"].(string)
		ns = strings.TrimSpace(ns)
		if ns == "" {
			ns = "default"
		}

		key := fmt.Sprintf("%s/%s/%s", kind, ns, name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, workloadRef{Kind: kind, Namespace: ns, Name: name})
	}

	return out
}

func (s *K8sWorkloadService) DeleteDeployment(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "Deployment")
}

func (s *K8sWorkloadService) DeleteStatefulSet(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "StatefulSet")
}

func (s *K8sWorkloadService) DeleteDaemonSet(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "DaemonSet")
}

func (s *K8sWorkloadService) DeleteJob(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "Job")
}

func (s *K8sWorkloadService) DeleteCronJob(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "CronJob")
}

func (s *K8sWorkloadService) deleteWorkloadByKind(ctx context.Context, req NamespacedDeleteRequest, kind string) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	gvk, ok := s.dyn.GVKByKind(kind)
	if !ok {
		return apperror.BadRequest("不支持的工作负载类型")
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 %s 失败: %v", kind, err))
	}
	return nil
}

func (s *K8sWorkloadService) CronJobSuspend(ctx context.Context, req CronJobSuspendRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("CronJob 不存在")
		}
		return apperror.Internal(fmt.Sprintf("获取 CronJob 失败: %v", err))
	}
	copyObj := obj.DeepCopy()
	copyObj.Spec.Suspend = &req.Suspend
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("更新 CronJob suspend 失败: %v", err))
	}
	return nil
}

func (s *K8sWorkloadService) CronJobTrigger(ctx context.Context, req CronJobTriggerRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	var cj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Name(req.Name).Get(&cj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return "", apperror.BadRequest("CronJob 不存在")
		}
		return "", apperror.Internal(fmt.Sprintf("获取 CronJob 失败: %v", err))
	}
	jobName := fmt.Sprintf("%s-manual-%d", cj.Name, time.Now().Unix())
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: req.Namespace,
			Labels:    cj.Labels,
		},
		Spec: cj.Spec.JobTemplate.Spec,
	}
	// keep a lightweight owner ref to cronjob (optional)
	job.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "batch/v1",
		Kind:       "CronJob",
		Name:       cj.Name,
		UID:        cj.UID,
	}}
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Create(job).Error; err != nil {
		return "", apperror.Internal(fmt.Sprintf("触发 CronJob 创建 Job 失败: %v", err))
	}
	return jobName, nil
}
