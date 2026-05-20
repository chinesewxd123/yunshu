package service

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func workloadContainerIndex(containers []corev1.Container, name string) int {
	n := strings.TrimSpace(name)
	if n != "" {
		for i, c := range containers {
			if c.Name == n {
				return i
			}
		}
		return -1
	}
	if len(containers) > 0 {
		return 0
	}
	return -1
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

func podTemplateResourceTotals(spec corev1.PodSpec) (reqCPU, reqMem, limCPU, limMem resource.Quantity) {
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
	return reqCPU, reqMem, limCPU, limMem
}

func podTemplateResourceSummary(spec corev1.PodSpec) string {
	reqCPU, reqMem, limCPU, limMem := podTemplateResourceTotals(spec)
	return fmt.Sprintf("CPU: %s / %s\n内存: %s / %s", quantityOrDash(reqCPU), quantityOrDash(limCPU), quantityOrDash(reqMem), quantityOrDash(limMem))
}

func workloadUsagePercents(u podCPUMemUsage, spec corev1.PodSpec, scale int64) (cpuUse, memUse string, cpuReqPct, cpuLimPct, memReqPct, memLimPct float64) {
	reqCPU, reqMem, limCPU, limMem := podTemplateResourceTotals(spec)
	if !u.CPU.IsZero() || !u.Mem.IsZero() {
		cpuUse = quantityOrDash(u.CPU)
		memUse = quantityOrDash(u.Mem)
	} else {
		cpuUse = "-"
		memUse = "-"
	}
	cpuReqPct = quantityPercentScaled(u.CPU, reqCPU, scale)
	cpuLimPct = quantityPercentScaled(u.CPU, limCPU, scale)
	memReqPct = quantityPercentScaled(u.Mem, reqMem, scale)
	memLimPct = quantityPercentScaled(u.Mem, limMem, scale)
	return cpuUse, memUse, cpuReqPct, cpuLimPct, memReqPct, memLimPct
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

// DeploymentDetail 执行对应的业务逻辑。
