package alertnotify

import (
	"fmt"
	"sort"
	"strings"
)

// auxiliaryHintKeys 常见补充维度（容器、主机磁盘、网络、中间件等），用于在正文突出一行「关键状态」。
var auxiliaryHintKeys = []string{
	"phase", "reason", "condition", "alertstate",
	"device", "mountpoint", "fstype", "model",
	"interface", "state", "volume",
}

// AuxiliaryStateHints 从 labels 提取少量高信号键值，逗号分隔。
func AuxiliaryStateHints(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	var parts []string
	for _, k := range auxiliaryHintKeys {
		if v := strings.TrimSpace(labels[k]); v != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(parts, ", ")
}

// FormatCompactLabels 将标签压缩为可读串（按 key 排序，最多 maxKeys 项）。
func FormatCompactLabels(labels map[string]string, maxKeys int) string {
	if labels == nil || len(labels) == 0 || maxKeys <= 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, maxKeys)
	for _, k := range keys {
		if len(parts) >= maxKeys {
			break
		}
		v := strings.TrimSpace(labels[k])
		if v == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ", ")
}
