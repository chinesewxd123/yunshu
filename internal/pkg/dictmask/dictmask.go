package dictmask

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// SensitiveDictType 判断字典类型是否应按「敏感凭据」脱敏（options 不落明文）。
func SensitiveDictType(dictType string) bool {
	t := strings.ToLower(strings.TrimSpace(dictType))
	if t == "" {
		return false
	}
	// 显名单类
	switch t {
	case "alert_webhook_token", "alert_enrich_prometheus_token",
		"minio_secret_key", "mail_password",
		"k8s_kubeconfig_template", "k8s_direct_config":
		return true
	}
	suffixes := []string{
		"_password", "_private_key", "_sk", "_secret", "_token",
		"_corp_secret", "_app_secret",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(t, s) {
			return true
		}
	}
	// Access Key 仍属高敏：options 仅脱敏预览
	if strings.HasSuffix(t, "_ak") {
		return true
	}
	return false
}

// Preview 返回用于下拉展示的脱敏文案（非可逆、不可用于业务回填）。
func Preview(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "（空）"
	}
	runes := []rune(s)
	n := len(runes)
	if n <= 8 {
		return "****"
	}
	head := string(runes[:4])
	tail := string(runes[n-4:])
	if strings.Contains(s, "BEGIN") && strings.Contains(s, "KEY") {
		return head + " …（" + strconv.Itoa(utf8.RuneCountInString(s)) + " 字符）… " + tail
	}
	return head + " … " + tail
}
