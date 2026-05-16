package service

import (
	"strings"

	"yunshu/internal/pkg/alertnotify"
)

// 分组节流 digest 默认额外纳入的对象维度（在 group_by 之外区分具体监控目标）。
var defaultTimingDigestBy = []string{
	"instance", "pod", "node", "host", "device", "mountpoint", "fqdn", "job",
}

// 平台注入或易变元数据，不参与「通知内容是否变化」判定。
var timingDigestExcludedKeys = map[string]struct{}{
	"fingerprint":       {},
	"datasource_id":     {},
	"datasource_name":   {},
	"datasource_type":   {},
	"monitor_pipeline":  {},
	"__name__":          {},
}

func isTimingDigestExcludedKey(k string) bool {
	k = strings.ToLower(strings.TrimSpace(k))
	if k == "" {
		return true
	}
	if _, ok := timingDigestExcludedKeys[k]; ok {
		return true
	}
	return strings.HasPrefix(k, "datasource_")
}

// resolveGroupByLabelValue 解析 group_by / digest_by 字段值（与 computeGroupKey 一致）。
func (s *AlertService) resolveGroupByLabelValue(field, receiver, status, severity, alertname string, labels map[string]string, dims alertnotify.Dims) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "alertname":
		return strings.TrimSpace(alertname)
	case "cluster":
		return strings.TrimSpace(dims.Cluster)
	case "namespace":
		return strings.TrimSpace(dims.Namespace)
	case "severity":
		return strings.TrimSpace(severity)
	case "receiver":
		return strings.TrimSpace(receiver)
	case "status":
		return strings.TrimSpace(status)
	default:
		if labels == nil {
			return ""
		}
		return strings.TrimSpace(labels[field])
	}
}

func (s *AlertService) timingDigestFieldKeys() []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 16)
	add := func(k string) {
		k = strings.ToLower(strings.TrimSpace(k))
		if isTimingDigestExcludedKey(k) {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	fields := s.cfg.GroupBy
	if len(fields) == 0 {
		fields = []string{"alertname", "cluster", "namespace", "severity", "receiver"}
	}
	for _, f := range fields {
		add(f)
	}
	extra := s.cfg.DigestBy
	if len(extra) == 0 {
		extra = defaultTimingDigestBy
	}
	for _, f := range extra {
		add(f)
	}
	return out
}

// labelsDigestForGroupTiming 方案 A：digest = hash(group_by 维度 + digest_by 对象维度)，不含全量 Prometheus labels。
func (s *AlertService) labelsDigestForGroupTiming(receiver, status, severity, alertname string, labels map[string]string) string {
	dims := alertnotify.ExtractDims(labels)
	pairs := make(map[string]string)
	for _, field := range s.timingDigestFieldKeys() {
		v := s.resolveGroupByLabelValue(field, receiver, status, severity, alertname, labels, dims)
		if v == "" {
			continue
		}
		pairs[field] = v
	}
	return alertnotify.DigestLabels(pairs)
}
