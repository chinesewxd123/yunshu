package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"

	"yunshu/internal/model"
)

func payloadMetaValueMeaningful(v interface{}) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		return s != "" && strings.EqualFold(s, "null") == false
	case map[string]interface{}:
		return len(x) > 0
	case []interface{}:
		return len(x) > 0
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		return s != "" && s != "<nil>" && strings.EqualFold(s, "null") == false
	}
}

func enrichRequestMapWithAlertPayload(reqMap map[string]interface{}, alertPayload map[string]interface{}) {
	if reqMap == nil {
		return
	}
	for _, key := range []string{"startsAt", "endsAt", "occurredAt", "generatorURL", "status", "severity", "groupKey", "monitorPipeline", "datasourceId", "datasourceName", "datasourceType", "current", "current_resolved"} {
		if existing, ok := reqMap[key]; ok && payloadMetaValueMeaningful(existing) {
			continue
		}
		if alertPayload == nil {
			continue
		}
		if v, ok := alertPayload[key]; ok && payloadMetaValueMeaningful(v) {
			reqMap[key] = v
		}
	}
	if existing, ok := reqMap["labels"]; ok && payloadMetaValueMeaningful(existing) {
		return
	}
	if alertPayload == nil {
		return
	}
	if v, ok := alertPayload["labels"]; ok && payloadMetaValueMeaningful(v) {
		reqMap["labels"] = v
	}
}

// shrinkLargestNotifyStrings 缩短钉钉/企微等大段 markdown，避免 json.Marshal 后按字节截断时丢掉排在后面的 startsAt 等字段。
func shrinkLargestNotifyStrings(m map[string]interface{}) bool {
	if md, ok := m["markdown"].(map[string]interface{}); ok {
		for _, key := range []string{"text", "content"} {
			s, ok := md[key].(string)
			if !ok || len(s) <= 512 {
				continue
			}
			cut := len(s) / 2
			if cut < 256 {
				cut = 256
			}
			md[key] = s[:cut] + "\n…(truncated)…"
			return true
		}
	}
	if tx, ok := m["text"].(map[string]interface{}); ok {
		s, ok := tx["content"].(string)
		if !ok || len(s) <= 512 {
			return false
		}
		cut := len(s) / 2
		if cut < 256 {
			cut = 256
		}
		tx["content"] = s[:cut] + "\n…(truncated)…"
		return true
	}
	// 钉钉应用会话：msg.markdown.text
	if msg, ok := m["msg"].(map[string]interface{}); ok {
		if md, ok := msg["markdown"].(map[string]interface{}); ok {
			if s, ok := md["text"].(string); ok && len(s) > 512 {
				cut := len(s) / 2
				if cut < 256 {
					cut = 256
				}
				md["text"] = s[:cut] + "\n…(truncated)…"
				return true
			}
		}
	}
	return false
}

func trimWebhookBodyForMaxJSON(m map[string]interface{}, maxBytes int) {
	if maxBytes <= 0 || m == nil {
		return
	}
	for iter := 0; iter < 64; iter++ {
		bs, err := json.Marshal(m)
		if err != nil || len(bs) <= maxBytes {
			return
		}
		if shrinkLargestNotifyStrings(m) {
			continue
		}
		if md, ok := m["markdown"].(map[string]interface{}); ok {
			title := md["title"]
			m["markdown"] = map[string]interface{}{
				"title": title,
				"text":  "[内容过长已省略，历史记录保留告警时间等字段]",
			}
			continue
		}
		if tx, ok := m["text"].(map[string]interface{}); ok {
			tx["content"] = "[内容过长已省略，历史记录保留告警时间等字段]"
			m["text"] = tx
			continue
		}
		if msg, ok := m["msg"].(map[string]interface{}); ok {
			if md, ok := msg["markdown"].(map[string]interface{}); ok {
				md["text"] = "[内容过长已省略，历史记录保留告警时间等字段]"
				msg["markdown"] = md
				m["msg"] = msg
				continue
			}
		}
		break
	}
}

func buildEventPayloadBytes(reqBytes []byte, alertPayload map[string]interface{}, maxChars int) []byte {
	// 历史记录展示需要 startsAt 等原始告警上下文（钉钉/企微下发体默认不包含），
	// 这里仅扩充入库 payload，不影响实际发往通道的请求体。
	if len(reqBytes) == 0 {
		return reqBytes
	}
	var reqMap map[string]interface{}
	if err := json.Unmarshal(reqBytes, &reqMap); err != nil {
		return reqBytes
	}
	if reqMap == nil {
		reqMap = map[string]interface{}{}
	}
	enrichRequestMapWithAlertPayload(reqMap, alertPayload)
	trimWebhookBodyForMaxJSON(reqMap, maxChars)
	bs, err := json.Marshal(reqMap)
	if err != nil {
		return reqBytes
	}
	return bs
}

func dingtalkRequestDebugNote(channel *model.AlertChannel, req *http.Request) string {
	if channel == nil || req == nil {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(channel.Type), "dingding") {
		return ""
	}
	rawURL := req.URL.String()
	masked := maskWebhookURL(rawURL)
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return "dingtalk debug: url=" + masked + " timestamp="
	}
	ts := strings.TrimSpace(parsed.Query().Get("timestamp"))
	return "dingtalk debug: url=" + masked + " timestamp=" + ts
}

func maskWebhookURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	parsed, err := neturl.Parse(raw)
	if err != nil {
		return raw
	}
	q := parsed.Query()
	for key := range q {
		lk := strings.ToLower(strings.TrimSpace(key))
		if lk == "sign" || strings.Contains(lk, "token") || strings.Contains(lk, "secret") || strings.Contains(lk, "access_token") {
			q.Set(key, "***")
		}
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

