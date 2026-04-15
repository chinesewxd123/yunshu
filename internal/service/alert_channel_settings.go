package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/parseutil"
	"go-permission-system/internal/pkg/validateutil"
)

func validateHeadersJSON(v string) error {
	return validateutil.ValidateJSONObjectString(v, "headers_json")
}

func requiresWebhookURL(channelType, headersJSON string) bool {
	t := strings.ToLower(strings.TrimSpace(channelType))
	if t == "email" {
		return false
	}
	if t == "dingding" {
		settings, err := parseChannelSettings(headersJSON)
		if err == nil {
			mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", settings["dingMode"])))
			if mode == "app_chat" {
				return false
			}
		}
	}
	return true
}

func parseChannelSettings(v string) (map[string]interface{}, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		return nil, apperror.BadRequest("headers_json 解析失败")
	}
	return m, nil
}

func parseRequestHeaders(settings map[string]interface{}) map[string]string {
	out := map[string]string{}
	if settings == nil {
		return out
	}
	for k, v := range settings {
		if strings.EqualFold(k, "to") || strings.EqualFold(k, "recipients") || strings.EqualFold(k, "emails") ||
			strings.EqualFold(k, "atMobiles") || strings.EqualFold(k, "atUserIds") || strings.EqualFold(k, "isAtAll") || strings.EqualFold(k, "headers") ||
			strings.EqualFold(k, "wecomMode") || strings.EqualFold(k, "corpID") || strings.EqualFold(k, "corpSecret") || strings.EqualFold(k, "agentId") ||
			strings.EqualFold(k, "dingMode") || strings.EqualFold(k, "appKey") || strings.EqualFold(k, "appSecret") || strings.EqualFold(k, "chatId") || strings.EqualFold(k, "signSecret") {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out[k] = s
		}
	}
	if hv, ok := settings["headers"]; ok {
		if hm, ok := hv.(map[string]interface{}); ok {
			for k, v := range hm {
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" {
					out[k] = s
				}
			}
		}
	}
	return out
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func truncateText(v string, n int) string {
	if n <= 0 || len(v) <= n {
		return v
	}
	return v[:n]
}

func validateEmailChannelRecipients(enabled bool, chType, headersJSON string) error {
	if !enabled || !strings.EqualFold(strings.TrimSpace(chType), "email") {
		return nil
	}
	recipients, err := parseEmailRecipients(headersJSON)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return apperror.BadRequest("email 通道需至少配置一个邮件接收人")
	}
	return nil
}

func splitEmailRecipientString(s string) []string {
	return validateutil.SplitRecipientString(s)
}

func normalizeRecipientList(v interface{}) []string {
	return validateutil.NormalizeRecipientList(v)
}

// parseEmailRecipients 从通道 HeadersJSON 解析收件人，兼容 to / recipients / emails（字符串、JSON 数组、逗号或分号分隔）。
func parseEmailRecipients(headersJSON string) ([]string, error) {
	headersJSON = strings.TrimSpace(headersJSON)
	if headersJSON == "" {
		return nil, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(headersJSON), &raw); err != nil {
		return nil, apperror.BadRequest("email 通道配置 JSON 解析失败")
	}
	seen := make(map[string]bool)
	var out []string
	for _, key := range []string{"to", "recipients", "emails"} {
		v, ok := raw[key]
		if !ok || v == nil {
			continue
		}
		list := normalizeRecipientList(v)
		for _, e := range list {
			e = strings.TrimSpace(e)
			if e == "" || seen[e] {
				continue
			}
			seen[e] = true
			out = append(out, e)
		}
	}
	return out, nil
}

func buildWebhookURL(channel *model.AlertChannel, settings map[string]interface{}, body []byte) string {
	base := strings.TrimSpace(channel.URL)
	if !strings.EqualFold(channel.Type, "dingding") {
		return base
	}
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", settings["dingMode"])))
	if mode == "app_chat" {
		return base
	}
	signSecret := strings.TrimSpace(fmt.Sprintf("%v", settings["signSecret"]))
	if signSecret == "" {
		signSecret = strings.TrimSpace(channel.Secret)
	}
	if signSecret == "" || base == "" {
		return base
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	toSign := ts + "\n" + signSecret
	mac := hmac.New(sha256.New, []byte(signSecret))
	_, _ = mac.Write([]byte(toSign))
	sign := neturl.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "timestamp=" + ts + "&sign=" + sign
}

func parseStringList(v interface{}) []string {
	return parseutil.ParseStringList(v)
}

func parseBool(v interface{}) bool {
	return parseutil.ParseBool(v)
}

func uniqueStrings(items []string) []string {
	return parseutil.UniqueStrings(items)
}

func maxInt(a, b int) int {
	return parseutil.MaxInt(a, b)
}
