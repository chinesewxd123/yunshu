package k8seventforward

import (
	"testing"

	"yunshu/internal/model"
)

func TestEventSeverity(t *testing.T) {
	if eventSeverity("Warning") != "warning" {
		t.Fatal("Warning")
	}
	if eventSeverity("Normal") != "info" {
		t.Fatal("Normal")
	}
}

func TestBuildAlertManagerPayload_ProjectID(t *testing.T) {
	p := buildAlertManagerPayload("r1", "1", "local", 42, []model.K8sForwardedEvent{
		{Type: "Warning", Reason: "Unhealthy", Namespace: "kube-system", Name: "Pod/x", Message: "probe failed"},
	})
	if len(p.Alerts) != 1 {
		t.Fatalf("alerts len %d", len(p.Alerts))
	}
	if p.Alerts[0].Labels["project_id"] != "42" {
		t.Fatalf("project_id=%q", p.Alerts[0].Labels["project_id"])
	}
	if p.Alerts[0].Labels["severity"] != "warning" {
		t.Fatalf("severity=%q", p.Alerts[0].Labels["severity"])
	}
}
