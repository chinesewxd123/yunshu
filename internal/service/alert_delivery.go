package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/alertnotify"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/parseutil"
)

type channelNotifyFunc func(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error)

func (s *AlertService) sendToChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
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
	groupKey := alertnotify.StringFromPayload(alertPayload, "group_key")
	labelsDigest := alertnotify.StringFromPayload(alertPayload, "labels_digest")
	if cluster == "" {
		cluster = alertnotify.StringFromPayload(body, "cluster")
	}
	if groupKey == "" {
		groupKey = alertnotify.StringFromPayload(body, "group_key")
	}
	if labelsDigest == "" {
		labelsDigest = alertnotify.StringFromPayload(body, "labels_digest")
	}
	event := model.AlertEvent{
		Source:          source,
		Title:           title,
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		GroupKey:        groupKey,
		LabelsDigest:    labelsDigest,
		ChannelID:       channel.ID,
		ChannelName:     channel.Name,
		Success:         reqErr == nil && code >= 200 && code < 300,
		HTTPStatusCode:  code,
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(respBody, s.cfg.MaxPayloadChars),
	}
	if reqErr != nil {
		event.ErrorMessage = truncateText(reqErr.Error(), 1000)
	} else if code < 200 || code >= 300 {
		event.ErrorMessage = truncateText(fmt.Sprintf("unexpected status code: %d", code), 1000)
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
	if reqErr != nil {
		return code, respBody, reqErr
	}
	if code < 200 || code >= 300 {
		return code, respBody, apperror.Internal(fmt.Sprintf("webhook 返回异常状态码: %d", code))
	}
	return code, respBody, nil
}

func (s *AlertService) sendEmailChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
	recipients, err := parseEmailRecipients(channel.HeadersJSON)
	if err != nil {
		return 0, "", err
	}
	if len(recipients) == 0 {
		return 0, "", apperror.BadRequest("email 通道未配置收件人：请在「邮件接收人」或配置 JSON 中填写 to / recipients / emails")
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return 0, "", apperror.Internal("邮件通道未配置：请检查全局 SMTP（mail 相关配置）是否启用")
	}
	subject := fmt.Sprintf("[告警][%s][%s] %s", strings.ToUpper(status), strings.ToUpper(severity), title)
	mdBody := alertnotify.RenderMarkdownCard(title, payload)
	htmlBody := alertnotify.MarkdownToHTML(mdBody)
	sendErr := error(nil)
	for _, to := range recipients {
		if err := s.mailer.SendMultipart(ctx, to, subject, mdBody, htmlBody); err != nil {
			sendErr = err
		}
	}
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          source,
		Title:           title,
		Severity:        severity,
		Status:          status,
		Cluster:         alertnotify.StringFromPayload(payload, "cluster"),
		GroupKey:        alertnotify.StringFromPayload(payload, "group_key"),
		LabelsDigest:    alertnotify.StringFromPayload(payload, "labels_digest"),
		ChannelID:       channel.ID,
		ChannelName:     channel.Name,
		Success:         sendErr == nil,
		HTTPStatusCode:  200,
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText("email sent", s.cfg.MaxPayloadChars),
	}
	if sendErr != nil {
		event.HTTPStatusCode = 500
		event.ErrorMessage = truncateText(sendErr.Error(), 1000)
		event.ResponsePayload = ""
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
	if sendErr != nil {
		return 500, "", sendErr
	}
	return 200, "email sent", nil
}
