package service

import (
	"context"
	"fmt"
	"strings"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var podMetricsGVK = schema.GroupVersionKind{Group: "metrics.k8s.io", Version: "v1beta1", Kind: "PodMetrics"}

// podCPUMemUsage 单 Pod 内各容器 usage 之和。
type podCPUMemUsage struct {
	CPU resource.Quantity
	Mem resource.Quantity
}

// nodeAllocResources 节点可分配 CPU/内存（用于 Pod 占用节点 alloc 占比）。
type nodeAllocResources struct {
	CPU resource.Quantity
	Mem resource.Quantity
}

func parseUnstructuredPodMetrics(u *unstructured.Unstructured) (namespace, podName string, cpu, mem resource.Quantity, ok bool) {
	ns, _, err1 := unstructured.NestedString(u.Object, "metadata", "namespace")
	name, _, err2 := unstructured.NestedString(u.Object, "metadata", "name")
	if err1 != nil || err2 != nil || ns == "" || name == "" {
		return "", "", cpu, mem, false
	}
	containers, found, err := unstructured.NestedSlice(u.Object, "containers")
	if !found || err != nil || len(containers) == 0 {
		return ns, name, cpu, mem, true
	}
	for _, c := range containers {
		cm, okm := c.(map[string]interface{})
		if !okm {
			continue
		}
		usage, _ := cm["usage"].(map[string]interface{})
		if usage == nil {
			continue
		}
		if v, okc := usage["cpu"].(string); okc && strings.TrimSpace(v) != "" {
			if q, err := resource.ParseQuantity(v); err == nil {
				cpu.Add(q)
			}
		}
		if v, okc := usage["memory"].(string); okc && strings.TrimSpace(v) != "" {
			if q, err := resource.ParseQuantity(v); err == nil {
				mem.Add(q)
			}
		}
	}
	return ns, name, cpu, mem, true
}

// listPodCPUMemUsageByNamespace 返回 namespace 内 Pod 名 -> 实时用量（需 metrics-server）。
func listPodCPUMemUsageByNamespace(ctx context.Context, dyn *DynamicResourceService, k *kom.Kubectl, namespace string) map[string]podCPUMemUsage {
	out := make(map[string]podCPUMemUsage)
	ns := strings.TrimSpace(namespace)
	if ns == "" || dyn == nil || k == nil {
		return out
	}
	items, err := dyn.ListByGVK(ctx, k, podMetricsGVK, ns)
	if err != nil {
		return out
	}
	for i := range items {
		_, podName, cpu, mem, ok := parseUnstructuredPodMetrics(&items[i])
		if !ok || podName == "" {
			continue
		}
		out[podName] = podCPUMemUsage{CPU: cpu, Mem: mem}
	}
	return out
}

// aggregatePodMetricsUsageByNamespace 全集群 PodMetrics，按命名空间汇总 CPU/Mem usage（一次 AllNamespace 列表）。
func aggregatePodMetricsUsageByNamespace(ctx context.Context, k *kom.Kubectl) map[string]podCPUMemUsage {
	out := make(map[string]podCPUMemUsage)
	if k == nil {
		return out
	}
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(podMetricsGVK)
	var list []unstructured.Unstructured
	if err := k.WithContext(ctx).Resource(u).AllNamespace().List(&list).Error; err != nil {
		return out
	}
	for i := range list {
		ns, _, cpu, mem, ok := parseUnstructuredPodMetrics(&list[i])
		if !ok || ns == "" {
			continue
		}
		agg := out[ns]
		agg.CPU.Add(cpu)
		agg.Mem.Add(mem)
		out[ns] = agg
	}
	return out
}

func maxQty(a, b resource.Quantity) resource.Quantity {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

// podSpecResourceTotals 工作负载 request/limit：普通容器求和；Init 容器按调度语义取各资源**最大值**再累加；Ephemeral 求和。
func podSpecResourceTotals(spec corev1.PodSpec) (reqCPU, reqMem, limCPU, limMem resource.Quantity) {
	for _, c := range spec.Containers {
		if c.Resources.Requests != nil {
			reqCPU.Add(c.Resources.Requests[corev1.ResourceCPU])
			reqMem.Add(c.Resources.Requests[corev1.ResourceMemory])
		}
		if c.Resources.Limits != nil {
			limCPU.Add(c.Resources.Limits[corev1.ResourceCPU])
			limMem.Add(c.Resources.Limits[corev1.ResourceMemory])
		}
	}
	var maxInitReqCPU, maxInitReqMem, maxInitLimCPU, maxInitLimMem resource.Quantity
	for _, c := range spec.InitContainers {
		if c.Resources.Requests != nil {
			maxInitReqCPU = maxQty(maxInitReqCPU, c.Resources.Requests[corev1.ResourceCPU])
			maxInitReqMem = maxQty(maxInitReqMem, c.Resources.Requests[corev1.ResourceMemory])
		}
		if c.Resources.Limits != nil {
			maxInitLimCPU = maxQty(maxInitLimCPU, c.Resources.Limits[corev1.ResourceCPU])
			maxInitLimMem = maxQty(maxInitLimMem, c.Resources.Limits[corev1.ResourceMemory])
		}
	}
	reqCPU.Add(maxInitReqCPU)
	reqMem.Add(maxInitReqMem)
	limCPU.Add(maxInitLimCPU)
	limMem.Add(maxInitLimMem)
	for _, ec := range spec.EphemeralContainers {
		if ec.Resources.Requests != nil {
			reqCPU.Add(ec.Resources.Requests[corev1.ResourceCPU])
			reqMem.Add(ec.Resources.Requests[corev1.ResourceMemory])
		}
		if ec.Resources.Limits != nil {
			limCPU.Add(ec.Resources.Limits[corev1.ResourceCPU])
			limMem.Add(ec.Resources.Limits[corev1.ResourceMemory])
		}
	}
	return reqCPU, reqMem, limCPU, limMem
}

func podResourceTotals(p corev1.Pod) (reqCPU, reqMem, limCPU, limMem resource.Quantity) {
	return podSpecResourceTotals(p.Spec)
}

// podContainersImageText 列表「容器/镜像」列：仅名称与镜像（资源在 resource_text 单独列展示）。
func podContainersImageText(p corev1.Pod) string {
	if len(p.Spec.Containers) == 0 {
		return "-"
	}
	var b strings.Builder
	for _, c := range p.Spec.Containers {
		img := strings.TrimSpace(c.Image)
		if img == "" {
			img = "-"
		}
		fmt.Fprintf(&b, "%s => %s\n", c.Name, img)
	}
	return strings.TrimSpace(b.String())
}

func podAggregatedResourceText(reqCPU, reqMem, limCPU, limMem resource.Quantity) string {
	return fmt.Sprintf("CPU req/limit %s / %s\n内存 req/limit %s / %s",
		quantityOrDash(reqCPU), quantityOrDash(limCPU), quantityOrDash(reqMem), quantityOrDash(limMem))
}

// aggregateDeploymentPodUsage PodMetrics 按 Deployment 聚合（经 ReplicaSet 归属）。
func aggregateDeploymentPodUsage(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace string) map[string]podCPUMemUsage {
	out := make(map[string]podCPUMemUsage)
	ns := strings.TrimSpace(namespace)
	if ns == "" || k == nil || dyn == nil {
		return out
	}
	usageByPod := listPodCPUMemUsageByNamespace(ctx, dyn, k, ns)
	var pods []corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).List(&pods).Error; err != nil {
		return out
	}
	var rss []appsv1.ReplicaSet
	if err := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(ns).List(&rss).Error; err != nil {
		return out
	}
	rsToDeploy := make(map[string]string, len(rss))
	for i := range rss {
		rs := &rss[i]
		for _, ref := range rs.OwnerReferences {
			if ref.Kind == "Deployment" && ref.Name != "" {
				// 多数集群会设置 controller=true；兼容旧数据或未标识 controller 的 RS
				if ref.Controller == nil || *ref.Controller {
					rsToDeploy[rs.Name] = ref.Name
					break
				}
			}
		}
	}
	for _, p := range pods {
		u := usageByPod[p.Name]
		if u.CPU.IsZero() && u.Mem.IsZero() {
			continue
		}
		for _, ref := range p.OwnerReferences {
			if ref.Kind != "ReplicaSet" || ref.Name == "" {
				continue
			}
			if ref.Controller != nil && !*ref.Controller {
				continue
			}
			dep := rsToDeploy[ref.Name]
			if dep == "" {
				continue
			}
			agg := out[dep]
			agg.CPU.Add(u.CPU)
			agg.Mem.Add(u.Mem)
			out[dep] = agg
			break
		}
	}
	return out
}

// aggregateStatefulSetPodUsage PodMetrics 按 StatefulSet 名称聚合。
func aggregateStatefulSetPodUsage(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace string) map[string]podCPUMemUsage {
	return aggregatePodUsageByOwnerKind(ctx, k, dyn, namespace, "StatefulSet")
}

// aggregateDaemonSetPodUsage PodMetrics 按 DaemonSet 名称聚合。
func aggregateDaemonSetPodUsage(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace string) map[string]podCPUMemUsage {
	return aggregatePodUsageByOwnerKind(ctx, k, dyn, namespace, "DaemonSet")
}

// aggregateJobPodUsage PodMetrics 按 Job 名称聚合（直接 controller OwnerReference）。
func aggregateJobPodUsage(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace string) map[string]podCPUMemUsage {
	return aggregatePodUsageByOwnerKind(ctx, k, dyn, namespace, "Job")
}

// aggregateCronJobPodUsage PodMetrics 按 CronJob 聚合（Pod -> Job -> CronJob owner）。
func aggregateCronJobPodUsage(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace string) map[string]podCPUMemUsage {
	out := make(map[string]podCPUMemUsage)
	ns := strings.TrimSpace(namespace)
	if ns == "" || k == nil || dyn == nil {
		return out
	}
	usageByPod := listPodCPUMemUsageByNamespace(ctx, dyn, k, ns)
	var jobs []batchv1.Job
	if err := k.WithContext(ctx).Resource(&batchv1.Job{}).Namespace(ns).List(&jobs).Error; err != nil {
		return out
	}
	jobParentCronJob := make(map[string]string, len(jobs))
	for i := range jobs {
		j := &jobs[i]
		for _, ref := range j.OwnerReferences {
			if ref.Kind == "CronJob" && ref.Name != "" {
				if ref.Controller == nil || *ref.Controller {
					jobParentCronJob[j.Name] = ref.Name
					break
				}
			}
		}
	}
	var pods []corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).List(&pods).Error; err != nil {
		return out
	}
	for _, p := range pods {
		u := usageByPod[p.Name]
		if u.CPU.IsZero() && u.Mem.IsZero() {
			continue
		}
		for _, ref := range p.OwnerReferences {
			if ref.Kind != "Job" || ref.Name == "" {
				continue
			}
			if ref.Controller != nil && !*ref.Controller {
				continue
			}
			cj := jobParentCronJob[ref.Name]
			if cj == "" {
				continue
			}
			agg := out[cj]
			agg.CPU.Add(u.CPU)
			agg.Mem.Add(u.Mem)
			out[cj] = agg
			break
		}
	}
	return out
}

func aggregatePodUsageByOwnerKind(ctx context.Context, k *kom.Kubectl, dyn *DynamicResourceService, namespace, kind string) map[string]podCPUMemUsage {
	out := make(map[string]podCPUMemUsage)
	ns := strings.TrimSpace(namespace)
	kind = strings.TrimSpace(kind)
	if ns == "" || kind == "" || k == nil || dyn == nil {
		return out
	}
	usageByPod := listPodCPUMemUsageByNamespace(ctx, dyn, k, ns)
	var pods []corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).List(&pods).Error; err != nil {
		return out
	}
	for _, p := range pods {
		u := usageByPod[p.Name]
		if u.CPU.IsZero() && u.Mem.IsZero() {
			continue
		}
		for _, ref := range p.OwnerReferences {
			if ref.Kind != kind || ref.Name == "" {
				continue
			}
			if ref.Controller != nil && !*ref.Controller {
				continue
			}
			name := ref.Name
			if name == "" {
				continue
			}
			agg := out[name]
			agg.CPU.Add(u.CPU)
			agg.Mem.Add(u.Mem)
			out[name] = agg
			break
		}
	}
	return out
}
