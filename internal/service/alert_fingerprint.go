package service

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// stableLabelsFingerprint 根据标签生成稳定指纹（与 Alertmanager fingerprint 语义兼容）。
func stableLabelsFingerprint(labels map[string]string) string {
	if fp := strings.TrimSpace(labels["fingerprint"]); fp != "" {
		return fp
	}
	if len(labels) == 0 {
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
		b.WriteByte('=')
		b.WriteString(strings.TrimSpace(labels[k]))
		b.WriteByte(';')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}
