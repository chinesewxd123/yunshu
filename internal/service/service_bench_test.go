package service

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/pkg/alertnotify"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func BenchmarkChannelMatchesAlert(b *testing.B) {
	settings := map[string]interface{}{
		"matchLabels": map[string]interface{}{"cluster": "prod"},
		"matchRegex":  map[string]interface{}{"namespace": "^kube-"},
	}
	labels := map[string]string{"env": "prod", "team": "sre"}
	dims := alertnotify.Dims{Cluster: "prod", Namespace: "kube-system"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		channelMatchesAlert(settings, labels, dims)
	}
}

func BenchmarkComputeGroupKey(b *testing.B) {
	s := &AlertService{cfg: config.AlertConfig{GroupBy: []string{"alertname", "cluster", "namespace", "severity", "receiver"}}}
	labels := map[string]string{
		"alertname": "HighCPU",
		"cluster":   "prod",
		"namespace": "app",
		"severity":  "warning",
	}
	dims := alertnotify.Dims{Cluster: "prod", Namespace: "app"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.computeGroupKey("webhook", "firing", "warning", "HighCPU", labels, dims)
	}
}

func BenchmarkShrinkLargestNotifyStrings(b *testing.B) {
	long := strings.Repeat("x", 4000)
	m := map[string]interface{}{
		"markdown": map[string]interface{}{"text": long},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := map[string]interface{}{
			"markdown": map[string]interface{}{"text": long},
		}
		shrinkLargestNotifyStrings(body)
		_ = body
		_ = m
	}
}

func BenchmarkWorkloadContainerIndex(b *testing.B) {
	containers := []corev1.Container{
		{Name: "c1"}, {Name: "c2"}, {Name: "c3"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workloadContainerIndex(containers, "c2")
	}
}

func BenchmarkDeploymentResourceSummary(b *testing.B) {
	cpu := resource.MustParse("250m")
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: cpu},
						},
					}},
				},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deploymentResourceSummary(d)
	}
}

func BenchmarkParseUintCSV(b *testing.B) {
	raw := "1,2,3,4,5,6,7,8,9,10"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseUintCSV(raw)
	}
}
