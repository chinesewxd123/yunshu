package service

import (
	"testing"
)

func TestPayloadString(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{"title": "  hello  "}
	if got := payloadString(m, "title"); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := payloadString(m, "missing"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveNotifyAlertName(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"labels": map[string]string{"alertname": "DiskFull"},
	}
	if got := resolveNotifyAlertName("", payload); got != "DiskFull" {
		t.Fatalf("got %q", got)
	}
	if got := resolveNotifyAlertName("", nil); got != "未命名告警" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectNameFromLabelsPayload(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"group_labels": map[string]interface{}{"project_name": "云枢"},
	}
	if got := projectNameFromLabelsPayload(payload); got != "云枢" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectIDFromPayload(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"labels": map[string]interface{}{"project_id": float64(99)},
	}
	if got := projectIDFromPayload(payload); got != 99 {
		t.Fatalf("got %d", got)
	}
	payload2 := map[string]interface{}{"project_id": uint(5)}
	if got := projectIDFromPayload(payload2); got != 5 {
		t.Fatalf("top-level: got %d", got)
	}
}

func TestParseUintAny(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   interface{}
		want uint
	}{
		{uint(3), 3},
		{int(7), 7},
		{float64(12), 12},
		{"15", 15},
		{"", 0},
		{nil, 0},
		{-1, 0},
	}
	for _, tc := range cases {
		if got := parseUintAny(tc.in); got != tc.want {
			t.Fatalf("%v: got %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestAppendAssigneePhonesToAtMobiles(t *testing.T) {
	t.Parallel()
	payload := map[string]interface{}{
		"assignee_phones": []string{"13800138000"},
	}
	out := appendAssigneePhonesToAtMobiles([]string{"13900139000"}, payload)
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}

func TestResolveNotifyProjectNameFallback(t *testing.T) {
	t.Parallel()
	s := &AlertService{}
	if got := s.resolveNotifyProjectName(nil, nil); got != "未绑定项目" {
		t.Fatalf("got %q", got)
	}
	payload := map[string]interface{}{"project_name": "P1"}
	if got := s.resolveNotifyProjectName(nil, payload); got != "P1" {
		t.Fatalf("got %q", got)
	}
}
