package alertnotify

import (
	"fmt"
	"strings"
	"time"
)

// StringFromPayload 读取 payload 中的字符串字段；缺失或非字符串化后为 <nil> 时返回空。
func StringFromPayload(payload map[string]interface{}, key string) string {
	if payload == nil || strings.TrimSpace(key) == "" {
		return ""
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

// PayloadLabels 从 webhook payload 中取出 labels map。
func PayloadLabels(payload map[string]interface{}) map[string]string {
	if payload == nil {
		return nil
	}
	switch m := payload["labels"].(type) {
	case map[string]string:
		return m
	case map[string]interface{}:
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		return out
	default:
		return nil
	}
}

// PayloadAnnotation 读取 annotations 中的键。
func PayloadAnnotation(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	switch m := payload["annotations"].(type) {
	case map[string]string:
		return strings.TrimSpace(m[key])
	case map[string]interface{}:
		return strings.TrimSpace(fmt.Sprintf("%v", m[key]))
	default:
		return ""
	}
}

// FormatPayloadTime 将 payload 中的时间字段格式化为本地可读字符串（兼容 time.Time / RFC3339 字符串）。
func FormatPayloadTime(v interface{}) string {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return ""
		}
		return t.In(time.Local).Format("2006-01-02 15:04:05 MST")
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			return parsed.In(time.Local).Format("2006-01-02 15:04:05 MST")
		}
		return s
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}

// ParsePayloadTime 解析 payload 时间为 time.Time。
func ParsePayloadTime(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return time.Time{}, false
		}
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, false
		}
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, false
		}
		return parsed, true
	default:
		return time.Time{}, false
	}
}
