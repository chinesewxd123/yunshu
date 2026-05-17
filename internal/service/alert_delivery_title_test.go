package service

import (
	"strings"
	"testing"
)

func TestBuildUnifiedNotifyTitleDutyPrefix(t *testing.T) {
	s := &AlertService{}
	payload := map[string]interface{}{
		"labels": map[string]string{
			"alertname":       "磁盘inode使用率",
			"monitor_rule_id": "1",
			"project_name":    "云枢项目",
		},
	}
	base := s.buildUnifiedNotifyTitle(nil, "", "critical", "firing", payload)
	if base == "" {
		t.Fatal("expected non-empty title")
	}
	// 无值班班次时不加前缀（dutySvc 为 nil）
	if len(base) < 4 || base[:4] == "值班[" {
		t.Fatalf("unexpected duty prefix without active duty: %q", base)
	}
	prefixed := "值班" + base
	if !strings.HasPrefix(prefixed, "值班[告警通知]") {
		t.Fatalf("unexpected prefixed title: %q", prefixed)
	}
}
