package alertnotify

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// DigestLabels 对 labels 做稳定排序后 SHA256，用于事件去重/分组摘要。
func DigestLabels(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
		b.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "sha256=" + hex.EncodeToString(sum[:])
}
