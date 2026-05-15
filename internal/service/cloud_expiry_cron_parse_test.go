package service

import (
	"testing"
	"time"
)

func TestParseCloudExpiryCronSchedule_everyMinute5Field(t *testing.T) {
	sched, err := parseCloudExpiryCronSchedule("*/1 * * * *")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t0 := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	n1 := sched.Next(t0)
	if !n1.After(t0) {
		t.Fatalf("expected next after t0, got %v -> %v", t0, n1)
	}
	if d := n1.Sub(t0); d < 30*time.Second || d > 2*time.Minute {
		t.Fatalf("expected ~1m step, got delta=%v (%v -> %v)", d, t0, n1)
	}
}
