package service

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/pkg/alertnotify"
)

func TestChannelMatchesAlert(t *testing.T) {
	t.Parallel()
	labels := map[string]string{"env": "prod", "team": "sre"}
	dims := alertnotify.Dims{Cluster: "c1", Namespace: "kube-system"}
	cases := []struct {
		name     string
		settings map[string]interface{}
		want     bool
	}{
		{"nil settings", nil, true},
		{"match labels ok", map[string]interface{}{
			"matchLabels": map[string]interface{}{"cluster": "c1", "env": "prod"},
		}, true},
		{"match labels fail", map[string]interface{}{
			"matchLabels": map[string]interface{}{"env": "staging"},
		}, false},
		{"match regex ok", map[string]interface{}{
			"matchRegex": map[string]interface{}{"namespace": "^kube-"},
		}, true},
		{"match regex fail", map[string]interface{}{
			"matchRegex": map[string]interface{}{"namespace": "^default$"},
		}, false},
		{"invalid regex", map[string]interface{}{
			"matchRegex": map[string]interface{}{"env": "[invalid"},
		}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := channelMatchesAlert(tc.settings, labels, dims); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestComputeGroupKeyStable(t *testing.T) {
	t.Parallel()
	s := &AlertService{cfg: config.AlertConfig{GroupBy: []string{"alertname", "cluster"}}}
	labels := map[string]string{"cluster": "prod", "alertname": "HighCPU"}
	dims := alertnotify.Dims{Cluster: "prod"}
	k1 := s.computeGroupKey("webhook", "firing", "critical", "HighCPU", labels, dims)
	k2 := s.computeGroupKey("webhook", "firing", "critical", "HighCPU", labels, dims)
	if k1 == "" || k1 != k2 {
		t.Fatalf("unstable keys: %q vs %q", k1, k2)
	}
	if !strings.HasPrefix(k1, "gk_") {
		t.Fatalf("expected gk_ prefix, got %q", k1)
	}
	k3 := s.computeGroupKey("webhook", "firing", "critical", "Other", labels, dims)
	if k1 == k3 {
		t.Fatalf("different alertname should differ")
	}
}

func TestParseLabelUint(t *testing.T) {
	t.Parallel()
	if n, ok := parseLabelUint("42"); !ok || n != 42 {
		t.Fatalf("42: n=%d ok=%v", n, ok)
	}
	if _, ok := parseLabelUint(""); ok {
		t.Fatal("empty should fail")
	}
	if _, ok := parseLabelUint("x"); ok {
		t.Fatal("invalid should fail")
	}
}

func TestParseLabelUintOrZero(t *testing.T) {
	t.Parallel()
	if parseLabelUintOrZero("7") != 7 {
		t.Fatal("expected 7")
	}
	if parseLabelUintOrZero("bad") != 0 {
		t.Fatal("expected 0")
	}
}

func TestMergeNotifyEmailsUnique(t *testing.T) {
	t.Parallel()
	out := mergeNotifyEmailsUnique([]string{"A@x.com", "a@x.com", " ", "b@x.com"})
	if len(out) != 2 {
		t.Fatalf("expected 2 unique, got %v", out)
	}
}

func TestMergeNotifyPhonesUnique(t *testing.T) {
	t.Parallel()
	out := mergeNotifyPhonesUnique([]string{"13800138000", "13800138000", "13900139000"})
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}
