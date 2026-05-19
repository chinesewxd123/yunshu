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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func (s *K8sWorkloadService) ListJobs(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Job")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt9987a1977622)
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
	jobUsage := aggregateJobPodUsage(ctx, k, s.dyn, q.Namespace)
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

		scale := int64(j.Status.Active)
		if scale < 1 {
			scale = 1
		}
		cpuUse, memUse, cr, cl, mr, ml := workloadUsagePercents(jobUsage[j.Name], j.Spec.Template.Spec, scale)

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
			CPUUsage:       cpuUse,
			MemUsage:       memUse,
			CPUPctRequest:  cr,
			CPUPctLimit:    cl,
			MemPctRequest:  mr,
			MemPctLimit:    ml,
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

// JobDetail 执行对应的业务逻辑。
func (s *K8sWorkloadService) JobDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("Job")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg656deb688b72)
		}
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt1a7e7f82dbdc)
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

// ListCronJobs 查询列表相关的业务逻辑。
func (s *K8sWorkloadService) JobRerun(ctx context.Context, req JobRerunRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	var job batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Name(req.Name).Get(&job).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return "", constants.ErrBadRequestWithMsg(constants.ErrMsg656deb688b72)
		}
		return "", svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt1a7e7f82dbdc)
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
		return "", svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt2abaeffc289e)
	}
	return newName, nil
}

func (s *K8sWorkloadService) JobPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var job batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(q.Namespace).Name(q.Name).Get(&job).Error; err != nil {
		return nil, svcerr.Internal(ctx, "k8s.workload", "api", err, constants.ErrFmt1a7e7f82dbdc)
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

// CronJobPods 执行对应的业务逻辑。
func (s *K8sWorkloadService) JobPatchContainerResources(ctx context.Context, req WorkloadContainerResourcesRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg656deb688b72)
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
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail(ctx, "k8s.workload", "api", err)
	}
	return nil
}

// CronJobPatchContainerResources 垂直扩缩：修改 CronJob 的 jobTemplate 内 Pod 模板资源。
// 与 Job 类似，批量/定时任务场景下变更模板主要影响后续创建的 Job/Pod；运行中实例不受影响。
