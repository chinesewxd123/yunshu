package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/parseutil"
)

func (s *AlertService) sendToChannel(ctx context.Context, channel *model.AlertChannel, env *alertdispatch.Envelope) (int, string, error) {
	if env == nil {
		env = alertdispatch.NewEnvelope("", "", "", "", map[string]interface{}{})
	}
	payload := env.PayloadOrEmpty()
	source := env.Source
	title := env.Title
	severity := env.Severity
	status := env.Status

	title = s.buildUnifiedNotifyTitle(ctx, title, severity, status, payload)
	payload["title"] = title
	settings, err := parseChannelSettings(channel.HeadersJSON)
	if err != nil {
		return 0, "", err
	}
	if strings.EqualFold(strings.TrimSpace(channel.Type), alertdispatch.ChannelTypeEmail) {
		return s.sendEmailChannel(ctx, channel, source, title, severity, status, payload)
	}
	typeKey := alertdispatch.NormalizeWebhookChannelType(channel.Type)
	notifyFn := s.buildNotifierRegistry()[typeKey]
	if notifyFn == nil {
		notifyFn = s.notifyGenericWebhook
	}
	return notifyFn(ctx, channel, source, title, severity, status, payload, settings)
}

func payloadString(payload map[string]interface{}, key string) string {
	return strings.TrimSpace(alertnotify.StringFromPayload(payload, key))
}

func (s *AlertService) renderChannelMessage(ctx context.Context, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) string {
	defaultMsg := alertnotify.RenderMarkdownCard(title, payload)
	if settings == nil {
		return defaultMsg
	}
	isResolved := strings.EqualFold(strings.TrimSpace(status), "resolved")
	tplRaw := strings.TrimSpace(fmt.Sprintf("%v", settings["messageTemplateFiring"]))
	if isResolved {
		tplRaw = strings.TrimSpace(fmt.Sprintf("%v", settings["messageTemplateResolved"]))
	}
	if tplRaw == "" || tplRaw == "<nil>" {
		return defaultMsg
	}
	projectName := s.resolveNotifyProjectName(ctx, payload)
	data := alertdispatch.BuildChannelTemplateData(title, severity, status, payload, projectName)
	tpl, err := template.New("channel_message").Option("missingkey=zero").Parse(tplRaw)
	if err != nil {
		return defaultMsg
	}
	var out bytes.Buffer
	if err = tpl.Execute(&out, data); err != nil {
		return defaultMsg
	}
	rendered := strings.TrimSpace(out.String())
	if rendered == "" {
		return defaultMsg
	}
	return rendered
}

func (s *AlertService) buildUnifiedNotifyTitle(ctx context.Context, rawTitle, severity, status string, payload map[string]interface{}) string {
	statusNorm := strings.ToLower(strings.TrimSpace(status))
	prefix := "告警通知"
	level := strings.ToUpper(strings.TrimSpace(severity))
	if statusNorm == "resolved" {
		prefix = "告警恢复"
		level = "RESOLVED"
	}
	if level == "" {
		level = strings.ToUpper(strings.TrimSpace(status))
	}
	if level == "" {
		level = "WARNING"
	}
	alertName := resolveNotifyAlertName(rawTitle, payload)
	projectName := s.resolveNotifyProjectName(ctx, payload)
	return fmt.Sprintf("[%s][%s][%s][%s]", prefix, level, projectName, alertName)
}

func resolveNotifyAlertName(rawTitle string, payload map[string]interface{}) string {
	if payload != nil {
		if labelsAny, ok := payload["labels"]; ok {
			switch labels := labelsAny.(type) {
			case map[string]string:
				if v := strings.TrimSpace(labels["alertname"]); v != "" {
					return v
				}
			case map[string]interface{}:
				if v := strings.TrimSpace(fmt.Sprintf("%v", labels["alertname"])); v != "" && v != "<nil>" {
					return v
				}
			}
		}
	}
	name := strings.TrimSpace(rawTitle)
	if name == "" {
		return "未命名告警"
	}
	return name
}

func projectNameFromLabelMap(labelsAny interface{}) string {
	switch labels := labelsAny.(type) {
	case map[string]string:
		return strings.TrimSpace(labels["project_name"])
	case map[string]interface{}:
		s := strings.TrimSpace(fmt.Sprintf("%v", labels["project_name"]))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	default:
		return ""
	}
}

func projectNameFromLabelsPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"labels", "group_labels"} {
		if labelsAny, ok := payload[key]; ok {
			if n := projectNameFromLabelMap(labelsAny); n != "" {
				return n
			}
		}
	}
	return ""
}

// projectIDFromPayload 从 payload 顶层、labels 或 group_labels 中解析 project_id（Alertmanager 有时只在 group 级别带标签）。
func projectIDFromPayload(payload map[string]interface{}) uint {
	if payload == nil {
		return 0
	}
	if id := parseUintAny(payload["project_id"]); id > 0 {
		return id
	}
	for _, key := range []string{"labels", "group_labels"} {
		if labelsAny, ok := payload[key]; ok {
			switch labels := labelsAny.(type) {
			case map[string]string:
				if id := parseUintAny(labels["project_id"]); id > 0 {
					return id
				}
			case map[string]interface{}:
				if id := parseUintAny(labels["project_id"]); id > 0 {
					return id
				}
			}
		}
	}
	return 0
}

func (s *AlertService) lookupProjectNameByID(ctx context.Context, id uint) string {
	if id == 0 || s.db == nil {
		return ""
	}
	var p model.Project
	if err := s.db.WithContext(ctx).Select("id", "name").First(&p, id).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(p.Name)
}

// enrichOutgoingProjectName 在发送前根据 project_id 写入 project_name，便于多渠道共用一次解析结果，并与告警历史 payload 一致。
func (s *AlertService) enrichOutgoingProjectName(ctx context.Context, payload map[string]interface{}) {
	if payload == nil {
		return
	}
	if name := strings.TrimSpace(fmt.Sprintf("%v", payload["project_name"])); name != "" && name != "<nil>" {
		return
	}
	if n := projectNameFromLabelsPayload(payload); n != "" {
		payload["project_name"] = n
		return
	}
	id := projectIDFromPayload(payload)
	if id == 0 {
		return
	}
	if n := s.lookupProjectNameByID(ctx, id); n != "" {
		payload["project_name"] = n
	}
}

func (s *AlertService) resolveNotifyProjectName(ctx context.Context, payload map[string]interface{}) string {
	const fallback = "未绑定项目"
	if payload == nil {
		return fallback
	}
	if name := strings.TrimSpace(fmt.Sprintf("%v", payload["project_name"])); name != "" && name != "<nil>" {
		return name
	}
	if name := projectNameFromLabelsPayload(payload); name != "" {
		return name
	}
	if id := projectIDFromPayload(payload); id > 0 {
		if n := s.lookupProjectNameByID(ctx, id); n != "" {
			return n
		}
	}
	return fallback
}

func parseUintAny(v interface{}) uint {
	switch vv := v.(type) {
	case uint:
		return vv
	case uint64:
		return uint(vv)
	case int:
		if vv > 0 {
			return uint(vv)
		}
		return 0
	case int64:
		if vv > 0 {
			return uint(vv)
		}
		return 0
	case float64:
		if vv > 0 {
			return uint(vv)
		}
		return 0
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "<nil>" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func (s *AlertService) notifyGenericWebhook(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error) {
	clamped := payload
	if s.cfg.PlatformLimits.GenericMaxChars > 0 {
		if m, ok := payload["summary"].(string); ok && runeLen(m) > s.cfg.PlatformLimits.GenericMaxChars {
			cp := map[string]interface{}{}
			for k, v := range payload {
				cp[k] = v
			}
			cp["summary"] = clampByRunes(m, s.cfg.PlatformLimits.GenericMaxChars)
			clamped = cp
		}
	}
	return s.postWebhookWithPayload(ctx, channel, source, title, severity, status, clamped, settings, payload)
}

func (s *AlertService) postWebhookWithPayloadMulti(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, bodies []map[string]interface{}, settings map[string]interface{}, alertPayload map[string]interface{}) (int, string, error) {
	if len(bodies) == 0 {
		return s.postWebhookWithPayload(ctx, channel, source, title, severity, status, map[string]interface{}{}, settings, alertPayload)
	}
	lastCode := 0
	lastResp := ""
	for _, b := range bodies {
		code, resp, err := s.postWebhookWithPayload(ctx, channel, source, title, severity, status, b, settings, alertPayload)
		lastCode, lastResp = code, resp
		if err != nil {
			return code, resp, err
		}
	}
	return lastCode, lastResp, nil
}

func (s *AlertService) notifyWeComWebhook(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error) {
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", settings["wecomMode"])))
	if mode == "" {
		mode = "robot"
	}
	if mode == "app" {
		return s.notifyWeComApp(ctx, channel, source, title, severity, status, payload, settings)
	}
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atUsers := parseutil.ParseStringList(settings["atUserIds"])
	if len(atMobiles) > 0 {
		resolved, _ := s.resolveWeComUserIDsByMobiles(ctx, settings, atMobiles)
		if len(resolved) > 0 {
			atUsers = append(atUsers, resolved...)
		}
	}
	atMobiles = parseutil.UniqueStrings(atMobiles)
	atUsers = parseutil.UniqueStrings(atUsers)
	message := s.renderChannelMessage(ctx, title, severity, status, payload, settings)
	outBody := buildWechatPayload(title, message, payload, settings, atMobiles, atUsers)
	bodies := splitWeComBody(outBody, s.cfg.PlatformLimits.WeComMaxChars)
	return s.postWebhookWithPayloadMulti(ctx, channel, source, title, severity, status, bodies, settings, payload)
}

func (s *AlertService) notifyDingTalkWebhook(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error) {
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", settings["dingMode"])))
	if mode == "" {
		mode = "robot"
	}
	if mode == "app_chat" {
		return s.notifyDingTalkAppChat(ctx, channel, source, title, severity, status, payload, settings)
	}
	atMobiles := parseutil.UniqueStrings(parseutil.ParseStringList(settings["atMobiles"]))
	atUsers := parseutil.UniqueStrings(parseutil.ParseStringList(settings["atUserIds"]))
	isAtAll := parseutil.ParseBool(settings["isAtAll"])
	message := s.renderChannelMessage(ctx, title, severity, status, payload, settings)
	outBody := buildDingTalkPayload(title, message, payload, settings, atMobiles, atUsers)
	bodies := splitDingTalkBody(outBody, s.cfg.PlatformLimits.DingdingMaxChars)
	if footer := atNotifyPlainMentionsFooter(atMobiles, atUsers, isAtAll); footer != "" && len(bodies) > 0 {
		appendDingTalkMarkdownText(bodies[len(bodies)-1], footer)
	}
	return s.postWebhookWithPayloadMulti(ctx, channel, source, title, severity, status, bodies, settings, payload)
}

func (s *AlertService) notifyWeComApp(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error) {
	corpID := strings.TrimSpace(fmt.Sprintf("%v", settings["corpID"]))
	corpSecret := strings.TrimSpace(fmt.Sprintf("%v", settings["corpSecret"]))
	agentID := strings.TrimSpace(fmt.Sprintf("%v", settings["agentId"]))
	if corpID == "" || corpSecret == "" || agentID == "" {
		return 0, "", apperror.BadRequest("企业微信应用模式需配置 corpID/corpSecret/agentId")
	}
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atUsers := parseutil.ParseStringList(settings["atUserIds"])
	if len(atMobiles) > 0 {
		resolved, _ := s.resolveWeComUserIDsByMobiles(ctx, settings, atMobiles)
		if len(resolved) > 0 {
			atUsers = append(atUsers, resolved...)
		}
	}
	atUsers = parseutil.UniqueStrings(atUsers)
	if parseutil.ParseBool(settings["isAtAll"]) {
		atUsers = append(atUsers, "@all")
	}
	if len(atUsers) == 0 {
		return 0, "", apperror.BadRequest("企业微信应用模式至少需要配置 atMobiles/atUserIds/isAtAll")
	}
	token, err := s.getWeComAccessToken(ctx, corpID, corpSecret)
	if err != nil {
		return 0, "", err
	}
	body := map[string]interface{}{
		"touser":  strings.Join(atUsers, "|"),
		"msgtype": "markdown",
		"agentid": agentID,
		"markdown": map[string]string{
			"content": alertnotify.RenderMarkdownCard(title, payload),
		},
		"safe": 0,
	}
	u := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + neturl.QueryEscape(token)
	return s.postDirect(ctx, source, title, severity, status, channel, u, body, map[string]string{}, payload)
}

func (s *AlertService) notifyDingTalkAppChat(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error) {
	appKey := strings.TrimSpace(fmt.Sprintf("%v", settings["appKey"]))
	appSecret := strings.TrimSpace(fmt.Sprintf("%v", settings["appSecret"]))
	chatID := strings.TrimSpace(fmt.Sprintf("%v", settings["chatId"]))
	if appKey == "" || appSecret == "" || chatID == "" {
		return 0, "", apperror.BadRequest("钉钉应用会话模式需配置 appKey/appSecret/chatId")
	}
	token, err := s.getDingTalkAccessToken(ctx, appKey, appSecret)
	if err != nil {
		return 0, "", err
	}
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atUsers := parseutil.ParseStringList(settings["atUserIds"])
	if len(atMobiles) > 0 {
		resolved, _ := s.resolveDingTalkUserIDsByMobiles(ctx, token, atMobiles)
		if len(resolved) > 0 {
			atUsers = append(atUsers, resolved...)
		}
	}
	body := map[string]interface{}{
		"chatid": chatID,
		"msg": map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": title,
				"text":  s.renderChannelMessage(ctx, title, severity, status, payload, settings),
			},
		},
		"at": map[string]interface{}{
			"atMobiles": parseutil.UniqueStrings(atMobiles),
			"atUserIds": parseutil.UniqueStrings(atUsers),
			"isAtAll":   parseutil.ParseBool(settings["isAtAll"]),
		},
	}
	u := "https://oapi.dingtalk.com/chat/send?access_token=" + neturl.QueryEscape(token)
	return s.postDirect(ctx, source, title, severity, status, channel, u, body, map[string]string{}, payload)
}

func (s *AlertService) postWebhookWithPayload(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, outBody map[string]interface{}, settings map[string]interface{}, alertPayload map[string]interface{}) (int, string, error) {
	reqBytes, _ := json.Marshal(outBody)
	headers := parseRequestHeaders(settings)
	url := buildWebhookURL(channel, settings, reqBytes)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return 0, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	if strings.TrimSpace(channel.Secret) != "" {
		sig := signBody(reqBytes, channel.Secret)
		httpReq.Header.Set("X-Webhook-Signature", sig)
	}
	return s.executeAndLogHTTP(ctx, source, title, severity, status, channel, outBody, alertPayload, reqBytes, httpReq)
}

func (s *AlertService) postDirect(ctx context.Context, source, title, severity, status string, channel *model.AlertChannel, url string, body map[string]interface{}, headers map[string]string, alertPayload map[string]interface{}) (int, string, error) {
	reqBytes, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return 0, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	return s.executeAndLogHTTP(ctx, source, title, severity, status, channel, body, alertPayload, reqBytes, httpReq)
}

func (s *AlertService) executeAndLogHTTP(ctx context.Context, source, title, severity, status string, channel *model.AlertChannel, body map[string]interface{}, alertPayload map[string]interface{}, reqBytes []byte, req *http.Request) (int, string, error) {
	timeout := maxInt(channel.TimeoutMS, s.cfg.DefaultTimeoutMS)
	client := &http.Client{Timeout: time.Duration(timeout) * time.Millisecond}
	resp, reqErr := client.Do(req)
	code := 0
	respBody := ""
	if resp != nil {
		code = resp.StatusCode
		bs, _ := io.ReadAll(resp.Body)
		respBody = string(bs)
		_ = resp.Body.Close()
	}

	cluster := alertnotify.StringFromPayload(alertPayload, "cluster")
	monPipe := strings.TrimSpace(alertnotify.StringFromPayload(alertPayload, "monitorPipeline"))
	groupKey := alertnotify.StringFromPayload(alertPayload, "groupKey")
	labelsDigest := alertnotify.StringFromPayload(alertPayload, "labelsDigest")
	if cluster == "" {
		cluster = alertnotify.StringFromPayload(body, "cluster")
	}
	if monPipe == "" {
		monPipe = strings.TrimSpace(alertnotify.StringFromPayload(body, "monitorPipeline"))
	}
	if groupKey == "" {
		groupKey = alertnotify.StringFromPayload(body, "groupKey")
	}
	if labelsDigest == "" {
		labelsDigest = alertnotify.StringFromPayload(body, "labelsDigest")
	}
	httpOK := reqErr == nil && code >= 200 && code < 300
	apiChecked, apiErr := alertdispatch.WebhookJSONAPIFailure(respBody)
	success := httpOK && (!apiChecked || apiErr == "")
	respPayload := truncateText(respBody, s.cfg.MaxPayloadChars)
	if dbg := dingtalkRequestDebugNote(channel, req); dbg != "" {
		if respPayload == "" {
			respPayload = truncateText(dbg, s.cfg.MaxPayloadChars)
		} else {
			respPayload = truncateText(dbg+"\n"+respPayload, s.cfg.MaxPayloadChars)
		}
	}
	event := model.AlertEvent{
		Source:             source,
		Title:              title,
		Severity:           severity,
		Status:             status,
		Cluster:            cluster,
		MonitorPipeline:    monPipe,
		GroupKey:           groupKey,
		LabelsDigest:       labelsDigest,
		MatchedPolicyIDs:   alertnotify.StringFromPayload(alertPayload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(alertPayload, "matchedPolicyNames"),
		ChannelID:          channel.ID,
		ChannelName:        channel.Name,
		Success:            success,
		HTTPStatusCode:     code,
		RequestPayload:     truncateText(string(buildEventPayloadBytes(reqBytes, alertPayload, s.cfg.MaxPayloadChars)), s.cfg.MaxPayloadChars),
		ResponsePayload:    respPayload,
	}
	if reqErr != nil {
		event.ErrorMessage = truncateText(reqErr.Error(), 1000)
	} else if code < 200 || code >= 300 {
		event.ErrorMessage = truncateText(fmt.Sprintf("unexpected status code: %d", code), 1000)
	} else if apiChecked && apiErr != "" {
		event.ErrorMessage = truncateText(apiErr, 1000)
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
	if reqErr != nil {
		return code, respBody, reqErr
	}
	if code < 200 || code >= 300 {
		return code, respBody, apperror.Internal(fmt.Sprintf("webhook 返回异常状态码: %d", code))
	}
	if apiChecked && apiErr != "" {
		return code, respBody, apperror.Internal(apiErr)
	}
	return code, respBody, nil
}

// payloadMetaValueMeaningful 判断入库 JSON 中某字段是否已有可用值（避免 webhook 体里占位空串阻断用 alertPayload 回填）。
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
	for _, key := range []string{"startsAt", "endsAt", "occurredAt", "generatorURL", "status", "severity", "groupKey", "monitorPipeline"} {
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

func mergeAssigneeEmails(recipients []string, payload map[string]interface{}) []string {
	// 严格处理人优先：
	// - assignee_emails 非空：仅发 assignee_emails（忽略通道固定收件人）
	// - assignee_emails 为空：使用通道固定收件人兜底
	var assignee []string
	if payload != nil {
		if raw, ok := payload["assignee_emails"]; ok {
			for _, e := range normalizeRecipientList(raw) {
				s := strings.TrimSpace(strings.ToLower(e))
				if s != "" {
					assignee = append(assignee, s)
				}
			}
		}
	}
	if len(assignee) > 0 {
		seen := map[string]struct{}{}
		out := make([]string, 0, len(assignee))
		for _, e := range assignee {
			if _, ok := seen[e]; ok {
				continue
			}
			seen[e] = struct{}{}
			out = append(out, e)
		}
		return out
	}

	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	for _, r := range recipients {
		add(r)
	}
	return out
}

func (s *AlertService) sendEmailChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
	recipients, err := parseEmailRecipients(channel.HeadersJSON)
	if err != nil {
		return 0, "", err
	}
	recipients = mergeAssigneeEmails(recipients, payload)
	if len(recipients) == 0 {
		return 0, "", apperror.BadRequest("邮件通道未配置收件人：请在邮件接收人或配置 JSON 中填写 to/recipients/emails；或由监控规则处理人 assignee_emails 提供")
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return 0, "", apperror.Internal("邮件通道未配置：请检查全局 SMTP（mail 相关配置）是否启用")
	}
	settings, err := parseChannelSettings(channel.HeadersJSON)
	if err != nil {
		return 0, "", err
	}
	subject := strings.TrimSpace(title)
	mdBody := s.renderChannelMessage(ctx, title, severity, status, payload, settings)
	htmlBody := alertnotify.MarkdownToHTML(mdBody)
	var failMsgs []string
	okCount := 0
	for _, to := range recipients {
		if err := s.mailer.SendMultipart(ctx, to, subject, mdBody, htmlBody); err != nil {
			failMsgs = append(failMsgs, fmt.Sprintf("%s: %v", to, err))
		} else {
			okCount++
		}
	}
	var sendErr error
	if okCount == 0 && len(recipients) > 0 {
		sendErr = fmt.Errorf("%s", strings.Join(failMsgs, "; "))
	} else if len(failMsgs) > 0 {
		sendErr = fmt.Errorf("partial failure: %s", strings.Join(failMsgs, "; "))
	}
	storeMap := make(map[string]interface{}, len(payload)+4)
	for k, v := range payload {
		storeMap[k] = v
	}
	storeMap["to"] = recipients
	alertdispatch.SlimOutgoingPayloadForHistory(storeMap, s.cfg.MaxPayloadChars)
	reqBytes, _ := json.Marshal(storeMap)
	respNote := "email sent"
	if okCount > 0 && len(failMsgs) > 0 {
		respNote = fmt.Sprintf("email sent: %d ok, %d failed", okCount, len(failMsgs))
	}
	event := model.AlertEvent{
		Source:             source,
		Title:              title,
		Severity:           severity,
		Status:             status,
		Cluster:            alertnotify.StringFromPayload(payload, "cluster"),
		MonitorPipeline:    strings.TrimSpace(alertnotify.StringFromPayload(payload, "monitorPipeline")),
		GroupKey:           alertnotify.StringFromPayload(payload, "groupKey"),
		LabelsDigest:       alertnotify.StringFromPayload(payload, "labelsDigest"),
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matchedPolicyNames"),
		ChannelID:          channel.ID,
		ChannelName:        channel.Name,
		Success:            okCount > 0,
		HTTPStatusCode:     200,
		RequestPayload:     truncateText(string(buildEventPayloadBytes(reqBytes, payload, s.cfg.MaxPayloadChars)), s.cfg.MaxPayloadChars),
		ResponsePayload:    truncateText(respNote, s.cfg.MaxPayloadChars),
	}
	if sendErr != nil && okCount == 0 {
		event.HTTPStatusCode = 500
		event.Success = false
		event.ErrorMessage = truncateText(sendErr.Error(), 1000)
		event.ResponsePayload = ""
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
	if okCount == 0 && sendErr != nil {
		return 500, "", sendErr
	}
	return 200, respNote, nil
}
