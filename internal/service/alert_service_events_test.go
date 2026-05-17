package service

import (
	"encoding/json"
	"testing"

	"yunshu/internal/model"
)

func TestExtractAlertIPFromPayload(t *testing.T) {
	t.Parallel()
	payload, _ := json.Marshal(map[string]interface{}{
		"labels": map[string]interface{}{"instance": "10.0.0.5:9100"},
	})
	if got := extractAlertIPFromPayload(string(payload), "fallback"); got != "10.0.0.5:9100" {
		t.Fatalf("got %q", got)
	}
	if got := extractAlertIPFromPayload("{}", "node-1"); got != "node-1" {
		t.Fatalf("fallback: got %q", got)
	}
}

func TestExtractAlertStartedAtFromPayload(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]interface{}{"startsAt": "2026-05-16T08:00:00Z"})
	if got := extractAlertStartedAtFromPayload(string(raw)); got != "2026-05-16T08:00:00Z" {
		t.Fatalf("got %q", got)
	}
}

func TestParseUintCSV(t *testing.T) {
	t.Parallel()
	got := parseUintCSV("1,2, 0 ,3,x,4")
	want := []uint{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %d want %d", i, got[i], want[i])
		}
	}
}

func TestPayloadValueByPath(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{
		"at": map[string]interface{}{
			"atMobiles": []interface{}{"13800138000"},
		},
	}
	v := payloadValueByPath(m, "at.atMobiles")
	list, ok := v.([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("got %v", v)
	}
}

func TestExtractEventReceiversEmail(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]interface{}{
		"assignee_emails": []string{"a@x.com", "B@x.com"},
	})
	out := extractEventReceivers(string(raw), "email-channel")
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}

func TestHydrateAlertEvent(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]interface{}{
		"labels":  map[string]interface{}{"cluster": "prod", "instance": "1.2.3.4"},
		"current": "85%",
	})
	ev := &model.AlertEvent{
		RequestPayload:     string(raw),
		Cluster:            "legacy-cluster",
		MatchedPolicyIDs:   "10,20",
		MatchedPolicyNames: "p1, p2",
		ChannelName:        "email",
	}
	hydrateAlertEvent(ev)
	if ev.Cluster != "prod" {
		t.Fatalf("cluster: %q", ev.Cluster)
	}
	if ev.AlertIP != "1.2.3.4" {
		t.Fatalf("alertIP: %q", ev.AlertIP)
	}
	if ev.MetricCurrent != "85%" {
		t.Fatalf("metric: %q", ev.MetricCurrent)
	}
	if len(ev.MatchedPolicyIDList) != 2 || ev.MatchedPolicyIDList[0] != 10 {
		t.Fatalf("policy ids: %v", ev.MatchedPolicyIDList)
	}
}

func TestFillMetricFieldsFromRequestPayload(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]interface{}{
		"current":          "1",
		"current_resolved": "0",
	})
	ev := &model.AlertEvent{RequestPayload: string(raw)}
	fillMetricFieldsFromRequestPayload(ev)
	if ev.MetricCurrent != "1" || ev.MetricResolved != "0" {
		t.Fatalf("current=%q resolved=%q", ev.MetricCurrent, ev.MetricResolved)
	}
}
