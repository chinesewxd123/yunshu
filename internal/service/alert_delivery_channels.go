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

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/parseutil"
	"yunshu/internal/service/svcerr"
)

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
			return code, resp, svcerr.Pass("alert.delivery", "postWebhookWithPayloadMulti", err)
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
	atMobiles = appendAssigneePhonesToAtMobiles(atMobiles, payload)
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
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atMobiles = appendAssigneePhonesToAtMobiles(atMobiles, payload)
	atMobiles = parseutil.UniqueStrings(atMobiles)
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
		return 0, "", constants.ErrBadRequestWithMsg(constants.ErrMsg5fcdf3f22c91)
	}
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atMobiles = appendAssigneePhonesToAtMobiles(atMobiles, payload)
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
		return 0, "", constants.ErrBadRequestWithMsg(constants.ErrMsg3963f2e4d87c)
	}
	token, err := s.getWeComAccessToken(ctx, corpID, corpSecret)
	if err != nil {
		return 0, "", svcerr.Pass("alert.delivery", "notifyWeComApp", err)
	}
	body := map[string]interface{}{
		"touser":  strings.Join(atUsers, "|"),
		"msgtype": "markdown",
		"agentid": agentID,
		"markdown": map[string]string{
			"content": s.renderChannelMessage(ctx, title, severity, status, payload, settings),
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
		return 0, "", constants.ErrBadRequestWithMsg(constants.ErrMsgf1768c17d51a)
	}
	token, err := s.getDingTalkAccessToken(ctx, appKey, appSecret)
	if err != nil {
		return 0, "", svcerr.Pass("alert.delivery", "notifyDingTalkAppChat", err)
	}
	atMobiles := parseutil.ParseStringList(settings["atMobiles"])
	atMobiles = appendAssigneePhonesToAtMobiles(atMobiles, payload)
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
		return 0, "", svcerr.Pass("alert.delivery", "postWebhookWithPayload", err)
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
		return 0, "", svcerr.Pass("alert.delivery", "postDirect", err)
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
	fillAlertEventDatasourceFromPayload(&event, alertPayload)
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
		return code, respBody, svcerr.InternalFmt("alert.delivery", "api", constants.ErrFmtd0ae16233479, code)
	}
	if apiChecked && apiErr != "" {
		return code, respBody, svcerr.InternalMsg("alert.delivery", "api", apiErr)
	}
	return code, respBody, nil
}

// payloadMetaValueMeaningful 判断入库 JSON 中某字段是否已有可用值（避免 webhook 体里占位空串阻断用 alertPayload 回填）。
