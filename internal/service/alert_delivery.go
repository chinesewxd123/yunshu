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
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/alertnotify"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/parseutil"
)

type channelNotifyFunc func(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error)

// webhookJSONAPIFailure 解析钉钉/企业微信等「HTTP 200 + JSON errcode」类响应。若存在 errcode 且非 0，返回具体错误文案；无 errcode 字段则视为非此类协议，不覆盖 HTTP 语义。
func webhookJSONAPIFailure(respBody string) (checked bool, errMsg string) {
	body := strings.TrimSpace(respBody)
	if len(body) == 0 || body[0] != '{' {
		return false, ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return false, ""
	}
	raw, ok := m["errcode"]
	if !ok {
		return false, ""
	}
	code := 0
	switch v := raw.(type) {
	case float64:
		code = int(v)
	case int:
		code = v
	case int64:
		code = int(v)
	case string:
		code, _ = strconv.Atoi(strings.TrimSpace(v))
	default:
		return false, ""
	}
	if code == 0 {
		return true, ""
	}
	msg := strings.TrimSpace(fmt.Sprintf("%v", m["errmsg"]))
	if msg == "" || msg == "<nil>" {
		msg = "errmsg empty"
	}
	return true, fmt.Sprintf("API errcode=%d: %s", code, msg)
}

func (s *AlertService) sendToChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
	title = s.buildUnifiedNotifyTitle(ctx, title, severity, payload)
	if payload != nil {
		payload["title"] = title
	}
	settings, err := parseChannelSettings(channel.HeadersJSON)
	if err != nil {
		return 0, "", err
	}
	if strings.EqualFold(channel.Type, "email") {
		return s.sendEmailChannel(ctx, channel, source, title, severity, status, payload)
	}
	typeKey := strings.ToLower(strings.TrimSpace(channel.Type))
	if typeKey == "" {
		typeKey = "generic_webhook"
	}
	notifyFn := s.buildNotifierRegistry()[typeKey]
	if notifyFn == nil {
		notifyFn = s.notifyGenericWebhook
	}
	return notifyFn(ctx, channel, source, title, severity, status, payload, settings)
}

func (s *AlertService) buildUnifiedNotifyTitle(ctx context.Context, rawTitle, severity string, payload map[string]interface{}) string {
	alarmLevel := strings.ToUpper(strings.TrimSpace(severity))
	if alarmLevel == "" {
		alarmLevel = "WARNING"
	}
	alertName := strings.TrimSpace(rawTitle)
	if alertName == "" {
		alertName = "未命名告警"
	}
	projectName := s.resolveNotifyProjectName(ctx, payload)
	return fmt.Sprintf("[告警][%s][%s][%s]", alarmLevel, projectName, alertName)
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

func (s *AlertService) buildNotifierRegistry() map[string]channelNotifyFunc {
	return map[string]channelNotifyFunc{
		"generic_webhook": s.notifyGenericWebhook,
		"wechat":          s.notifyWeComWebhook,
		"wechat_work":     s.notifyWeComWebhook,
		"dingding":        s.notifyDingTalkWebhook,
	}
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
	outBody := buildWechatPayload(title, payload, settings, atMobiles, atUsers)
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
	outBody := buildDingTalkPayload(title, payload, settings, atMobiles, atUsers)
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
				"text":  alertnotify.RenderMarkdownCard(title, payload),
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
	monPipe := strings.TrimSpace(alertnotify.StringFromPayload(alertPayload, "monitor_pipeline"))
	groupKey := alertnotify.StringFromPayload(alertPayload, "group_key")
	labelsDigest := alertnotify.StringFromPayload(alertPayload, "labels_digest")
	if cluster == "" {
		cluster = alertnotify.StringFromPayload(body, "cluster")
	}
	if monPipe == "" {
		monPipe = strings.TrimSpace(alertnotify.StringFromPayload(body, "monitor_pipeline"))
	}
	if groupKey == "" {
		groupKey = alertnotify.StringFromPayload(body, "group_key")
	}
	if labelsDigest == "" {
		labelsDigest = alertnotify.StringFromPayload(body, "labels_digest")
	}
	httpOK := reqErr == nil && code >= 200 && code < 300
	apiChecked, apiErr := webhookJSONAPIFailure(respBody)
	success := httpOK && (!apiChecked || apiErr == "")
	event := model.AlertEvent{
		Source:             source,
		Title:              title,
		Severity:           severity,
		Status:             status,
		Cluster:            cluster,
		MonitorPipeline:    monPipe,
		GroupKey:           groupKey,
		LabelsDigest:       labelsDigest,
		MatchedPolicyIDs:   alertnotify.StringFromPayload(alertPayload, "matched_policy_ids"),
		MatchedPolicyNames: alertnotify.StringFromPayload(alertPayload, "matched_policy_names"),
		ChannelID:          channel.ID,
		ChannelName:        channel.Name,
		Success:            success,
		HTTPStatusCode:     code,
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload:    truncateText(respBody, s.cfg.MaxPayloadChars),
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

func mergeAssigneeEmails(recipients []string, payload map[string]interface{}) []string {
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
	if payload != nil {
		if raw, ok := payload["assignee_emails"]; ok {
			for _, e := range normalizeRecipientList(raw) {
				add(e)
			}
		}
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
	subject := strings.TrimSpace(title)
	mdBody := alertnotify.RenderMarkdownCard(title, payload)
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
	reqBytes, _ := json.Marshal(payload)
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
		MonitorPipeline:    strings.TrimSpace(alertnotify.StringFromPayload(payload, "monitor_pipeline")),
		GroupKey:           alertnotify.StringFromPayload(payload, "group_key"),
		LabelsDigest:       alertnotify.StringFromPayload(payload, "labels_digest"),
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matched_policy_ids"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matched_policy_names"),
		ChannelID:          channel.ID,
		ChannelName:        channel.Name,
		Success:            okCount > 0,
		HTTPStatusCode:     200,
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
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
