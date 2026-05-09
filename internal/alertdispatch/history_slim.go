package alertdispatch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prometheus/K8s 等告警 labels 极大；JSON 按字典序序列化时 to 往往排在 labels/summary 之后，
// 若仅做字节截断会裁掉 to，导致历史「接收人」为空。落库前压缩 labels/annotations 等大字段。
var historyRetentionLabelKeys = []string{
	"alertname", "severity", "namespace", "pod", "pod_name", "instance", "node", "cluster",
	"project_id", "job", "deployment", "container", "prometheus", "region", "service",
	"app", "name", "statefulset", "daemonset", "workload", "uid",
	"datasource_id", "datasource_name", "datasource_type", "yunshu_datasource_id", "monitor_pipeline",
}

func truncateHistoryString(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func toStringInterfaceMap(raw interface{}) (map[string]interface{}, bool) {
	switch t := raw.(type) {
	case map[string]interface{}:
		return t, true
	case map[string]string:
		out := make(map[string]interface{}, len(t))
		for k, v := range t {
			out[k] = v
		}
		return out, true
	default:
		return nil, false
	}
}

func slimHistoryMapField(m map[string]interface{}, field string, priorityKeys []string, maxKeys int, maxValLen int, minJSONBytes int) bool {
	raw, ok := m[field]
	if !ok || raw == nil {
		return false
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil || len(rawBytes) < minJSONBytes {
		return false
	}
	sm, ok := toStringInterfaceMap(raw)
	if !ok || len(sm) == 0 {
		return false
	}
	out := make(map[string]interface{})
	seen := map[string]struct{}{}
	for _, pk := range priorityKeys {
		if v, ok := sm[pk]; ok {
			out[pk] = truncateHistoryString(fmt.Sprintf("%v", v), maxValLen)
			seen[pk] = struct{}{}
			if len(out) >= maxKeys {
				break
			}
		}
	}
	for k, v := range sm {
		if len(out) >= maxKeys {
			break
		}
		if _, ok := seen[k]; ok {
			continue
		}
		out[k] = truncateHistoryString(fmt.Sprintf("%v", v), maxValLen)
		seen[k] = struct{}{}
	}
	if len(sm) > len(out) {
		out["_source_key_count"] = len(sm)
	}
	m[field] = out
	return true
}

func truncateHistoryTopString(m map[string]interface{}, key string, max int) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || len(s) <= max {
		return false
	}
	m[key] = truncateHistoryString(s, max)
	return true
}

func wipeHeavyJSONField(m map[string]interface{}, key string, minBytes int) bool {
	raw, ok := m[key]
	if !ok || raw == nil {
		return false
	}
	bs, err := json.Marshal(raw)
	if err != nil || len(bs) < minBytes {
		return false
	}
	m[key] = map[string]interface{}{"_trimmed": true, "_approx_bytes": len(bs)}
	return true
}

func forceDropLabelsForHistory(m map[string]interface{}) bool {
	raw, ok := m["labels"]
	if !ok || raw == nil {
		return false
	}
	bs, err := json.Marshal(raw)
	if err != nil || len(bs) < 200 {
		return false
	}
	m["labels"] = map[string]interface{}{"_trimmed": true, "_approx_bytes": len(bs)}
	return true
}

// SlimOutgoingPayloadForHistory 在写入 alert_events.request_payload 前压缩大字段，
// 避免 JSON 过长截断时丢失字典序靠后的 to/startsAt 等关键键。
func SlimOutgoingPayloadForHistory(m map[string]interface{}, maxBytes int) {
	if maxBytes <= 0 || m == nil {
		return
	}
	for iter := 0; iter < 120; iter++ {
		bs, err := json.Marshal(m)
		if err != nil || len(bs) <= maxBytes {
			return
		}
		if slimHistoryMapField(m, "labels", historyRetentionLabelKeys, 36, 200, 1800) {
			continue
		}
		if slimHistoryMapField(m, "labels", historyRetentionLabelKeys, 14, 96, 700) {
			continue
		}
		if slimHistoryMapField(m, "labels", historyRetentionLabelKeys, 8, 64, 200) {
			continue
		}
		if slimHistoryMapField(m, "annotations", []string{"summary", "description"}, 8, 900, 1200) {
			continue
		}
		if truncateHistoryTopString(m, "summary", 800) {
			continue
		}
		if truncateHistoryTopString(m, "title", 240) {
			continue
		}
		if truncateHistoryTopString(m, "generatorURL", 600) {
			continue
		}
		if truncateHistoryTopString(m, "matchedPolicyNames", 480) {
			continue
		}
		if truncateHistoryTopString(m, "matchedPolicyIds", 320) {
			continue
		}
		if wipeHeavyJSONField(m, "group_labels", 1200) {
			continue
		}
		if forceDropLabelsForHistory(m) {
			continue
		}
		for _, k := range []string{"annotations", "truncated", "am_version", "agg_first_seen", "agg_last_seen"} {
			if _, ok := m[k]; ok {
				delete(m, k)
				break
			}
		}
	}
}
