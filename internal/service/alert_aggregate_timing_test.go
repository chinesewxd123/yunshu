package service

import (
	"testing"
	"time"

	"yunshu/internal/config"
)

func TestEvaluateFiringGroupTiming_groupInterval(t *testing.T) {
	cfg := config.AlertConfig{GroupWaitSeconds: 0, GroupIntervalSeconds: 60, RepeatIntervalSeconds: 300}
	now := time.Date(2026, 5, 16, 9, 25, 0, 0, time.UTC)
	lastSent := now.Add(-30 * time.Second).Format(time.RFC3339)
	should, reason := evaluateFiringGroupTiming(cfg, now, "", lastSent, "sha256=aaa", "sha256=bbb")
	if should || reason != "group_interval_suppressed" {
		t.Fatalf("expected group_interval_suppressed, got send=%v reason=%q", should, reason)
	}
	lastSentOld := now.Add(-90 * time.Second).Format(time.RFC3339)
	should, reason = evaluateFiringGroupTiming(cfg, now, "", lastSentOld, "sha256=aaa", "sha256=bbb")
	if !should || reason != "" {
		t.Fatalf("expected allow after 90s, got send=%v reason=%q", should, reason)
	}
}

func TestEvaluateFiringGroupTiming_repeatInterval(t *testing.T) {
	cfg := config.AlertConfig{GroupIntervalSeconds: 60, RepeatIntervalSeconds: 300}
	now := time.Date(2026, 5, 16, 9, 25, 0, 0, time.UTC)
	lastSent := now.Add(-120 * time.Second).Format(time.RFC3339)
	digest := "sha256=same"
	should, reason := evaluateFiringGroupTiming(cfg, now, "", lastSent, digest, digest)
	if should || reason != "repeat_suppressed" {
		t.Fatalf("expected repeat_suppressed, got send=%v reason=%q", should, reason)
	}
	lastSentOld := now.Add(-400 * time.Second).Format(time.RFC3339)
	should, reason = evaluateFiringGroupTiming(cfg, now, "", lastSentOld, digest, digest)
	if !should || reason != "" {
		t.Fatalf("expected allow after repeat window, got send=%v reason=%q", should, reason)
	}
}
