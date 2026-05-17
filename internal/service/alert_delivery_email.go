package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
)

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

func payloadHasAssigneeEmails(payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}
	raw, ok := payload["assignee_emails"]
	if !ok || raw == nil {
		return false
	}
	for _, e := range normalizeRecipientList(raw) {
		if strings.TrimSpace(e) != "" {
			return true
		}
	}
	return false
}

// mergeAssigneeEmailsWithReceiverGroup 合并接收组静态抄送；已有规则处理人邮箱时不合并（避免多人收件）。
func mergeAssigneeEmailsWithReceiverGroup(recipients []string, payload map[string]interface{}) []string {
	if payload == nil || payloadHasAssigneeEmails(payload) {
		return recipients
	}
	raw, ok := payload["receiver_group_emails"]
	if !ok || raw == nil {
		return recipients
	}
	var extra []string
	for _, e := range normalizeRecipientList(raw) {
		s := strings.TrimSpace(strings.ToLower(e))
		if s != "" {
			extra = append(extra, s)
		}
	}
	if len(extra) == 0 {
		return recipients
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	for _, r := range recipients {
		add(r)
	}
	for _, e := range extra {
		add(e)
	}
	return out
}

func (s *AlertService) sendEmailChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
	recipients, err := parseEmailRecipients(channel.HeadersJSON)
	if err != nil {
		return 0, "", svcerr.Pass("alert.delivery", "sendEmailChannel", err)
	}
	recipients = mergeAssigneeEmails(recipients, payload)
	recipients = mergeAssigneeEmailsWithReceiverGroup(recipients, payload)
	if len(recipients) == 0 {
		return 0, "", constants.ErrBadRequestWithMsg(constants.ErrMsgc47e8ed41463)
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return 0, "", svcerr.InternalMsg("alert.delivery", "api", constants.ErrMsg71c5fe1e9994)
	}
	settings, err := parseChannelSettings(channel.HeadersJSON)
	if err != nil {
		return 0, "", svcerr.Pass("alert.delivery", "sendEmailChannel", err)
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
	fillAlertEventDatasourceFromPayload(&event, payload)
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
