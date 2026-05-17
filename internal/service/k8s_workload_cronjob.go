package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sutil"
	"yunshu/internal/service/svcerr"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func (s *K8sWorkloadService) ListCronJobs(ctx context.Context, q NamespacedListQuery) ([]WorkloadItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt336d54b211b0)
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

// ListCronJobsV2 查询列表相关的业务逻辑。
func (s *K8sWorkloadService) ListCronJobsV2(ctx context.Context, q NamespacedListQuery) ([]CronJobItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	listU, err := s.dyn.ListByGVKWithSelector(ctx, k, gvk, q.Namespace, q.LabelQuery)
	if err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt336d54b211b0)
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
	cjUsage := aggregateCronJobPodUsage(ctx, k, s.dyn, q.Namespace)
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
		scale := int64(len(cj.Status.Active))
		if scale < 1 {
			scale = 1
		}
		tpl := cj.Spec.JobTemplate.Spec.Template.Spec
		cpuUse, memUse, cr, cl, mr, ml := workloadUsagePercents(cjUsage[cj.Name], tpl, scale)

		out = append(out, CronJobItem{
			Name:               cj.Name,
			Namespace:          cj.Namespace,
			Schedule:           cj.Spec.Schedule,
			Suspend:            suspend,
			ReadyPercent:       100,
			ResourceText:       podTemplateResourceSummary(tpl),
			ContainersText:     podTemplateContainersSummary(tpl),
			ConditionsText:     "-",
			Labels:             cj.Labels,
			Annotations:        cj.Annotations,
			LastScheduleTime:   last,
			LastSuccessfulTime: lastSuccess,
			ActiveCount:        activeCount,
			CPUUsage:           cpuUse,
			MemUsage:           memUse,
			CPUPctRequest:      cr,
			CPUPctLimit:        cl,
			MemPctRequest:      mr,
			MemPctLimit:        ml,
			CreationTime:       cj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Age:                k8sutil.HumanAge(cj.CreationTimestamp.Time),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// JobRerun 执行对应的业务逻辑。
func (s *K8sWorkloadService) CronJobPods(ctx context.Context, q RelatedPodsQuery) ([]RelatedPodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var cj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(q.Namespace).Name(q.Name).Get(&cj).Error; err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt687b79e3dfdb)
	}

	// CronJob 触发后，Pod 通常由 Job 创建并带有 job-name 标签；
	// 直接按 cronjob 标签查 Pod 在部分版本/场景会为空，因此改为“先找 Job，再找 Pod”。
	var jobs []batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(q.Namespace).List(&jobs).Error; err != nil {
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmtc13b046a7597)
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

func (s *K8sWorkloadService) CronJobDetail(ctx context.Context, q NamespacedDetailQuery) (*WorkloadDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, _ := s.dyn.GVKByKind("CronJob")
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc6ae960d40d1)
		}
		return nil, svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt687b79e3dfdb)
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

// Apply 提交申请相关的业务逻辑。
func (s *K8sWorkloadService) CronJobPatchContainerResources(ctx context.Context, req WorkloadContainerResourcesRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var obj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgc6ae960d40d1)
		}
		return k8sFail("k8s.workload", "api", err)
	}
	containers := obj.Spec.JobTemplate.Spec.Template.Spec.Containers
	idx := workloadContainerIndex(containers, req.ContainerName)
	if idx < 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg1a5aaa6cfa35)
	}
	copyObj := obj.DeepCopy()
	c := &copyObj.Spec.JobTemplate.Spec.Template.Spec.Containers[idx]
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
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail("k8s.workload", "api", err)
	}
	return nil
}
