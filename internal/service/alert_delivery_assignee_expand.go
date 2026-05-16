package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
)

func isCriticalAlertSeverity(payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", payload["severity"])), "critical")
}

func monitorRuleIDFromPayload(payload map[string]interface{}) (uint, bool) {
	if payload == nil {
		return 0, false
	}
	if raw, ok := payload["labels"].(map[string]string); ok && raw != nil {
		if id, ok2 := parseLabelUint(raw["monitor_rule_id"]); ok2 {
			return id, true
		}
	}
	if raw, ok := payload["labels"].(map[string]interface{}); ok && raw != nil {
		if id, ok2 := parseLabelUint(fmt.Sprintf("%v", raw["monitor_rule_id"])); ok2 {
			return id, true
		}
	}
	return 0, false
}

// resolveCriticalMailFallbackEmails 仅规则处理人显式用户/额外邮箱 + 当前值班邮箱，不含部门子树展开，也不含项目全员。
func (s *AlertService) resolveCriticalMailFallbackEmails(ctx context.Context, ruleID uint, status string) []string {
	var emails []string
	if s.assigneeSvc != nil && (status != "resolved" || s.assigneeSvc.NotifyOnResolvedEnabled(ctx, ruleID)) {
		e, _ := s.assigneeSvc.ResolveNotifyEmailsDirectUsers(ctx, ruleID)
		emails = append(emails, e...)
	}
	if s.dutySvc != nil {
		e, _ := s.dutySvc.ResolveNotifyEmailsAtRule(ctx, ruleID, time.Now())
		emails = append(emails, e...)
	}
	return mergeNotifyEmailsUnique(emails)
}

// expandChannelSetForAssigneeNotification critical 且接收组仅 IM 通道时，补启邮件通道并仅向规则处理人（显式用户）邮箱发信。
// 钉钉侧仍用 payload 内 assignee_phones 做 @；与是否在群内无关。
func (s *AlertService) expandChannelSetForAssigneeNotification(
	ctx context.Context,
	channelSet map[uint]struct{},
	receiverGroupIDs []uint,
	payload map[string]interface{},
) {
	if s == nil || len(channelSet) == 0 || len(receiverGroupIDs) == 0 || payload == nil {
		return
	}
	if !isCriticalAlertSeverity(payload) {
		return
	}
	status := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", payload["status"])))
	if status == "" {
		status = "firing"
	}
	if s.receiverGroupCache == nil {
		return
	}

	hasEmailChannelInGroups := false
	var rgEmails []string
	for _, gid := range receiverGroupIDs {
		g, err := s.receiverGroupCache.Get(gid)
		if err != nil || g == nil || !g.IsActiveNow() {
			continue
		}
		rgEmails = append(rgEmails, g.EmailRecipients...)
		for _, cid := range g.ChannelIDs {
			if cid == 0 {
				continue
			}
			if s.isEmailChannelID(ctx, cid) {
				hasEmailChannelInGroups = true
				channelSet[cid] = struct{}{}
			}
		}
	}

	// 接收组已配置邮件通道：沿用 enrich 后的 assignee_emails，可合并接收组静态抄送
	if hasEmailChannelInGroups {
		assignee := collectEmailsFromPayload(payload, "assignee_emails")
		if len(rgEmails) > 0 && len(assignee) > 0 {
			payload["assignee_emails"] = mergeNotifyEmailsUnique(append(assignee, rgEmails...))
		}
		return
	}

	// 仅钉钉/企微等：补邮件通道，收件人仅限规则处理人显式邮箱（非项目全员、不展开处理人部门子树）
	ruleID, ok := monitorRuleIDFromPayload(payload)
	if !ok || ruleID == 0 {
		return
	}
	fallback := s.resolveCriticalMailFallbackEmails(ctx, ruleID, status)
	if len(fallback) == 0 {
		return
	}
	payload["assignee_emails"] = fallback
	payload["assignee_mail_fallback_critical"] = true

	hasEmailInSet := false
	for cid := range channelSet {
		if s.isEmailChannelID(ctx, cid) {
			hasEmailInSet = true
			break
		}
	}
	if !hasEmailInSet {
		if id := s.firstEnabledEmailChannelID(ctx); id > 0 {
			channelSet[id] = struct{}{}
		}
	}
}

func collectEmailsFromPayload(payload map[string]interface{}, key string) []string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	var out []string
	for _, e := range normalizeRecipientList(raw) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e != "" {
			out = append(out, e)
		}
	}
	return mergeNotifyEmailsUnique(out)
}

func (s *AlertService) isEmailChannelID(ctx context.Context, id uint) bool {
	if id == 0 || s == nil {
		return false
	}
	var ch model.AlertChannel
	if err := s.db.WithContext(ctx).Select("type").First(&ch, id).Error; err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(ch.Type), alertdispatch.ChannelTypeEmail)
}

func (s *AlertService) firstEnabledEmailChannelID(ctx context.Context) uint {
	if s == nil || s.db == nil {
		return 0
	}
	var ch model.AlertChannel
	err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Where("LOWER(TRIM(type)) = ?", alertdispatch.ChannelTypeEmail).
		Order("id ASC").
		First(&ch).Error
	if err != nil {
		return 0
	}
	return ch.ID
}
