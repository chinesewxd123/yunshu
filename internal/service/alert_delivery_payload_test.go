package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPayloadMetaValueMeaningful(t *testing.T) {
	t.Parallel()
	if payloadMetaValueMeaningful(nil) {
		t.Fatal("nil not meaningful")
	}
	if !payloadMetaValueMeaningful("ok") {
		t.Fatal("string should be meaningful")
	}
	if payloadMetaValueMeaningful("null") || payloadMetaValueMeaningful("  ") {
		t.Fatal("null/space not meaningful")
	}
	if !payloadMetaValueMeaningful(map[string]interface{}{"a": 1}) {
		t.Fatal("map meaningful")
	}
}

func TestEnrichRequestMapWithAlertPayload(t *testing.T) {
	t.Parallel()
	req := map[string]interface{}{"title": "t"}
	alert := map[string]interface{}{
		"groupKey": "gk1",
		"labels":   map[string]string{"k": "v"},
	}
	enrichRequestMapWithAlertPayload(req, alert)
	if req["groupKey"] != "gk1" {
		t.Fatalf("groupKey: %v", req["groupKey"])
	}
	if req["labels"] == nil {
		t.Fatal("expected labels merged")
	}
}

func TestShrinkLargestNotifyStrings(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 600)
	m := map[string]interface{}{
		"markdown": map[string]interface{}{"text": long},
	}
	if !shrinkLargestNotifyStrings(m) {
		t.Fatal("expected shrink")
	}
	md := m["markdown"].(map[string]interface{})
	text := md["text"].(string)
	if len(text) >= len(long) {
		t.Fatalf("not shrunk: len=%d", len(text))
	}
}

func TestTrimWebhookBodyForMaxJSON(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{
		"markdown": map[string]interface{}{
			"title": "t",
			"text":  strings.Repeat("a", 8000),
		},
		"startsAt": "2026-05-16T00:00:00Z",
	}
	trimWebhookBodyForMaxJSON(m, 2048)
	bs, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) > 2048+256 {
		t.Fatalf("still too large: %d bytes", len(bs))
	}
	if !strings.Contains(string(bs), "startsAt") {
		t.Fatal("should keep startsAt after trim")
	}
}

func TestMaskWebhookURL(t *testing.T) {
	t.Parallel()
	masked := maskWebhookURL("https://oapi.dingtalk.com/robot/send?access_token=SECRET123")
	if strings.Contains(masked, "SECRET123") {
		t.Fatalf("token leaked: %q", masked)
	}
	if !strings.Contains(masked, "access_token=") {
		t.Fatalf("unexpected: %q", masked)
	}
}

func TestBuildEventPayloadBytes(t *testing.T) {
	t.Parallel()
	req := []byte(`{"a":1}`)
	alert := map[string]interface{}{"startsAt": "t0"}
	out := buildEventPayloadBytes(req, alert, 4096)
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["startsAt"] != "t0" {
		t.Fatalf("got %v", m["startsAt"])
	}
}
