package service

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/service/svcerr"
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
		return 0, "", svcerr.Pass(ctx, "alert.delivery", "sendToChannel", err)
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

func appendAssigneePhonesToAtMobiles(atMobiles []string, payload map[string]interface{}) []string {
	if payload == nil {
		return atMobiles
	}
	raw, ok := payload["assignee_phones"]
	if !ok || raw == nil {
		return atMobiles
	}
	for _, p := range normalizeRecipientList(raw) {
		p = strings.TrimSpace(p)
		if p != "" {
			atMobiles = append(atMobiles, p)
		}
	}
	return atMobiles
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
	title := fmt.Sprintf("[%s][%s][%s][%s]", prefix, level, projectName, alertName)
	if s.shouldPrefixDutyOnNotifyTitle(ctx, payload) {
		title = "值班" + title
	}
	return title
}

func (s *AlertService) shouldPrefixDutyOnNotifyTitle(ctx context.Context, payload map[string]interface{}) bool {
	rid, ok := monitorRuleIDFromPayload(payload)
	if !ok || rid == 0 || s.dutySvc == nil {
		return false
	}
	active, err := s.dutySvc.HasActiveBlockAtRule(ctx, rid, time.Now())
	return err == nil && active
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

