package service

import (
	"testing"

	"yunshu/internal/config"
)

func TestValidateWebhookToken(t *testing.T) {
	t.Parallel()
	s := &AlertService{cfg: config.AlertConfig{WebhookToken: "secret-token"}}
	if !s.ValidateWebhookToken("secret-token") {
		t.Fatal("exact match")
	}
	if !s.ValidateWebhookToken("Bearer secret-token") {
		t.Fatal("bearer prefix")
	}
	if s.ValidateWebhookToken("wrong") {
		t.Fatal("should reject wrong token")
	}
	if s.ValidateWebhookToken("") {
		t.Fatal("empty client token")
	}
	emptyCfg := &AlertService{cfg: config.AlertConfig{WebhookToken: ""}}
	if emptyCfg.ValidateWebhookToken("anything") {
		t.Fatal("server empty token must reject all")
	}
}

func TestMergeStringMap(t *testing.T) {
	t.Parallel()
	base := map[string]string{"a": "1", "b": "2"}
	override := map[string]string{"b": "9", "c": "3"}
	out := mergeStringMap(base, override)
	if out["a"] != "1" || out["b"] != "9" || out["c"] != "3" {
		t.Fatalf("got %v", out)
	}
	if base["b"] != "2" {
		t.Fatal("base should not mutate")
	}
}

func TestAlertEventSourceFromPayload(t *testing.T) {
	t.Parallel()
	if got := alertEventSourceFromPayload(map[string]interface{}{"source": "cloud_expiry"}); got != "cloud_expiry" {
		t.Fatalf("got %q", got)
	}
	if got := alertEventSourceFromPayload(nil); got != "alertmanager" {
		t.Fatalf("default %q", got)
	}
}

func TestMonitorPipelineFromPayload(t *testing.T) {
	t.Parallel()
	if got := monitorPipelineFromPayload(map[string]interface{}{"monitorPipeline": "ds:1"}); got != "ds:1" {
		t.Fatalf("got %q", got)
	}
	if got := monitorPipelineFromPayload(map[string]interface{}{}); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestPayloadUintAny(t *testing.T) {
	t.Parallel()
	if payloadUintAny(float64(3)) != 3 {
		t.Fatal("float64")
	}
	if payloadUintAny("5") != 5 {
		t.Fatal("string")
	}
	if payloadUintAny(nil) != 0 {
		t.Fatal("nil")
	}
}
