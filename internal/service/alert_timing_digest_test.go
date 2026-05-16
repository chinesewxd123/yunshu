package service

import (
	"testing"

	"yunshu/internal/config"
)

func TestLabelsDigestForGroupTiming_ignoresVolatileExtraLabels(t *testing.T) {
	s := &AlertService{cfg: config.AlertConfig{
		GroupBy:  []string{"alertname", "severity", "receiver"},
		DigestBy: []string{"instance"},
	}}
	labels := map[string]string{
		"alertname":     "服务器内存使用率",
		"severity":      "critical",
		"instance":      "172.16.0.10:9273",
		"job":           "node-exporter",
		"random_scrape": "will-change-every-eval",
		"fingerprint":   "should-be-excluded",
		"datasource_id": "99",
	}
	d1 := s.labelsDigestForGroupTiming("webhook", "firing", "critical", "服务器内存使用率", labels)
	labels["random_scrape"] = "another-value"
	labels["fingerprint"] = "other-fp"
	d2 := s.labelsDigestForGroupTiming("webhook", "firing", "critical", "服务器内存使用率", labels)
	if d1 == "" || d1 != d2 {
		t.Fatalf("digest should be stable when only excluded/extra labels change: %q vs %q", d1, d2)
	}
}

func TestLabelsDigestForGroupTiming_changesOnInstance(t *testing.T) {
	s := &AlertService{cfg: config.AlertConfig{
		GroupBy:  []string{"alertname", "severity"},
		DigestBy: []string{"instance"},
	}}
	base := map[string]string{"alertname": "CPU使用率", "severity": "warning", "instance": "10.0.0.1:9100"}
	d1 := s.labelsDigestForGroupTiming("webhook", "firing", "warning", "CPU使用率", base)
	base["instance"] = "10.0.0.2:9100"
	d2 := s.labelsDigestForGroupTiming("webhook", "firing", "warning", "CPU使用率", base)
	if d1 == d2 {
		t.Fatal("digest should change when instance changes")
	}
}

func TestLabelsDigestForGroupTiming_changesOnAlertname(t *testing.T) {
	s := &AlertService{cfg: config.AlertConfig{
		GroupBy:  []string{"alertname", "severity"},
		DigestBy: []string{"instance"},
	}}
	labels := map[string]string{"severity": "critical", "instance": "172.16.0.10:9273"}
	dMem := s.labelsDigestForGroupTiming("webhook", "firing", "critical", "服务器内存使用率", labels)
	dCPU := s.labelsDigestForGroupTiming("webhook", "firing", "critical", "CPU使用率", labels)
	if dMem == dCPU {
		t.Fatal("different alertname should yield different digest")
	}
}
