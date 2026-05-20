package service

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestWorkloadContainerIndex(t *testing.T) {
	t.Parallel()
	containers := []corev1.Container{
		{Name: "app", Image: "nginx"},
		{Name: "sidecar", Image: "busybox"},
	}
	if got := workloadContainerIndex(containers, "sidecar"); got != 1 {
		t.Fatalf("by name: got %d", got)
	}
	if got := workloadContainerIndex(containers, ""); got != 0 {
		t.Fatalf("empty name defaults first: got %d", got)
	}
	if got := workloadContainerIndex(containers, "missing"); got != -1 {
		t.Fatalf("missing: got %d", got)
	}
	if got := workloadContainerIndex(nil, ""); got != -1 {
		t.Fatalf("no containers: got %d", got)
	}
}

func TestDeploymentResourceSummary(t *testing.T) {
	t.Parallel()
	cpu := resource.MustParse("100m")
	mem := resource.MustParse("128Mi")
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "nginx",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    cpu,
								corev1.ResourceMemory: mem,
							},
						},
					}},
				},
			},
		},
	}
	s := deploymentResourceSummary(d)
	if !strings.Contains(s, "100m") || !strings.Contains(s, "128Mi") {
		t.Fatalf("unexpected summary: %q", s)
	}
}

func TestDeploymentContainersSummary(t *testing.T) {
	t.Parallel()
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "nginx:1.25"}},
				},
			},
		},
	}
	s := deploymentContainersSummary(d)
	if s != "app: nginx:1.25" {
		t.Fatalf("got %q", s)
	}
}

func TestDeploymentConditionsSummary(t *testing.T) {
	t.Parallel()
	d := appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			},
		},
	}
	s := deploymentConditionsSummary(d)
	if !strings.Contains(s, "Available=True") {
		t.Fatalf("got %q", s)
	}
}

func TestWorkloadUsagePercents(t *testing.T) {
	t.Parallel()
	spec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		}},
	}
	u := podCPUMemUsage{CPU: resource.MustParse("50m"), Mem: resource.MustParse("50Mi")}
	cpuUse, memUse, cr, _, mr, _ := workloadUsagePercents(u, spec, 2)
	if cpuUse == "-" || memUse == "-" {
		t.Fatalf("expected usage strings, cpu=%s mem=%s", cpuUse, memUse)
	}
	if cr <= 0 || mr <= 0 {
		t.Fatalf("expected positive pct, cr=%v mr=%v", cr, mr)
	}
}

func TestJobConditionsSummary(t *testing.T) {
	t.Parallel()
	j := batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			},
		},
	}
	s := jobConditionsSummary(j)
	if !strings.Contains(s, "Complete") {
		t.Fatalf("got %q", s)
	}
}
