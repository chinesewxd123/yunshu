package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sutil"
	"yunshu/internal/service/svcerr"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (s *K8sWorkloadService) Apply(ctx context.Context, req NamespacedApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
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
		return k8sFail("k8s.workload", "api", err)
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

// DeleteDeployment 删除相关的业务逻辑。
func (s *K8sWorkloadService) DeleteDeployment(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "Deployment")
}

// DeleteStatefulSet 删除相关的业务逻辑。
func (s *K8sWorkloadService) DeleteStatefulSet(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "StatefulSet")
}

// DeleteDaemonSet 删除相关的业务逻辑。
func (s *K8sWorkloadService) DeleteDaemonSet(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "DaemonSet")
}

// DeleteJob 删除相关的业务逻辑。
func (s *K8sWorkloadService) DeleteJob(ctx context.Context, req NamespacedDeleteRequest) error {
	return s.deleteWorkloadByKind(ctx, req, "Job")
}

// DeleteCronJob 删除相关的业务逻辑。
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
		return constants.ErrBadRequestWithMsg(constants.ErrMsgd5692b195622)
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return svcerr.Internalf("k8s.workload", "delete", err, constants.ErrFmt32b88f9cc2e5, kind)
	}
	return nil
}

// CronJobSuspend 执行对应的业务逻辑。
func (s *K8sWorkloadService) CronJobSuspend(ctx context.Context, req CronJobSuspendRequest) error {
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
	copyObj := obj.DeepCopy()
	copyObj.Spec.Suspend = &req.Suspend
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return k8sFail("k8s.workload", "api", err)
	}
	return nil
}

// CronJobTrigger 执行对应的业务逻辑。
func (s *K8sWorkloadService) CronJobTrigger(ctx context.Context, req CronJobTriggerRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	var cj batchv1.CronJob
	if err := k.WithContext(ctx).Resource(&batchv1.CronJob{}).Namespace(req.Namespace).Name(req.Name).Get(&cj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return "", constants.ErrBadRequestWithMsg(constants.ErrMsgc6ae960d40d1)
		}
		return "", svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt687b79e3dfdb)
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
		return "", svcerr.Internal("k8s.workload", "api", err, constants.ErrFmt25f9e144a662)
	}
	return jobName, nil
}

// DaemonSetPatchContainerResources 垂直扩缩：修改 DaemonSet Pod 模板内指定容器的 requests/limits。
// VPA 虽可纳管 DaemonSet，但 DaemonSet 按节点全局副本运行，资源上调可能导致节点压力，运维上需谨慎评估。
