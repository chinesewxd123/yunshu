package service

import (
	"testing"
)

func TestIsCriticalAlertSeverity(t *testing.T) {
	if !isCriticalAlertSeverity(map[string]interface{}{"severity": "critical"}) {
		t.Fatal("expected critical")
	}
	if isCriticalAlertSeverity(map[string]interface{}{"severity": "warning"}) {
		t.Fatal("expected non-critical")
	}
}

func TestMonitorRuleIDFromPayload(t *testing.T) {
	id, ok := monitorRuleIDFromPayload(map[string]interface{}{
		"labels": map[string]string{"monitor_rule_id": "42"},
	})
	if !ok || id != 42 {
		t.Fatalf("expected rule 42, got %d ok=%v", id, ok)
	}
}
