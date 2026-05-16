package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
)

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

type channelKindFlags struct {
	hasEmail    bool
	hasDingding bool
	hasWeCom    bool
	hasWechat   bool
}

func channelKindFromType(t string) string {
	return alertdispatch.NormalizeWebhookChannelType(t)
}

func (s *AlertService) channelKindFlagsForSet(ctx context.Context, channelSet map[uint]struct{}) channelKindFlags {
	var f channelKindFlags
	if s == nil || s.db == nil || len(channelSet) == 0 {
		return f
	}
	for cid := range channelSet {
		var ch model.AlertChannel
		if err := s.db.WithContext(ctx).Select("type").First(&ch, cid).Error; err != nil {
			continue
		}
		switch channelKindFromType(ch.Type) {
		case alertdispatch.ChannelTypeEmail:
			f.hasEmail = true
		case alertdispatch.ChannelTypeDingding:
			f.hasDingding = true
		case alertdispatch.ChannelTypeWechatWork:
			f.hasWeCom = true
		case alertdispatch.ChannelTypeWechat:
			f.hasWechat = true
		}
	}
	return f
}

func collectAssigneePhonesFromPayload(payload map[string]interface{}) []string {
	raw, ok := payload["assignee_phones"]
	if !ok || raw == nil {
		return nil
	}
	var out []string
	for _, p := range normalizeRecipientList(raw) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return mergeNotifyPhonesUnique(out)
}

// assigneePhoneResolvableOnDingWecom 手机号能否在钉钉/企微企业通讯录解析到（无法解析则通常无法 @，需邮件兜底）。
func (s *AlertService) assigneePhoneResolvableOnDingWecom(ctx context.Context, channelSet map[uint]struct{}, phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" || s == nil || s.db == nil {
		return false
	}
	for cid := range channelSet {
		var ch model.AlertChannel
		if err := s.db.WithContext(ctx).First(&ch, cid).Error; err != nil {
			continue
		}
		settings, err := parseChannelSettings(ch.HeadersJSON)
		if err != nil {
			continue
		}
		switch channelKindFromType(ch.Type) {
		case alertdispatch.ChannelTypeDingding:
			appKey := strings.TrimSpace(fmt.Sprintf("%v", settings["appKey"]))
			appSecret := strings.TrimSpace(fmt.Sprintf("%v", settings["appSecret"]))
			if appKey == "" || appSecret == "" {
				continue
			}
			token, err := s.getDingTalkAccessToken(ctx, appKey, appSecret)
			if err != nil || token == "" {
				continue
			}
			uid, err := s.getDingTalkUserIDByMobile(ctx, token, phone)
			if err == nil && strings.TrimSpace(uid) != "" {
				return true
			}
		case alertdispatch.ChannelTypeWechatWork:
			resolved, err := s.resolveWeComUserIDsByMobiles(ctx, settings, []string{phone})
			if err == nil && len(resolved) > 0 {
				return true
			}
		}
	}
	return false
}

func (s *AlertService) anyAssigneePhoneNeedsMailFallback(ctx context.Context, channelSet map[uint]struct{}, phones []string) bool {
	if len(phones) == 0 {
		return true
	}
	for _, p := range phones {
		if !s.assigneePhoneResolvableOnDingWecom(ctx, channelSet, p) {
			return true
		}
	}
	return false
}

// assigneeShouldReceiveSupplementalEmail 是否补启邮件通道向处理人发信。
// - wechat 等：始终补邮件；
// - 钉钉/企微：处理人无手机号，或手机号无法在企业通讯录解析（通常不在群内无法 @）时补邮件；
// - 接收组已含邮件通道：不重复补启通道，但仍写入 assignee_emails 供邮件投递。
func (s *AlertService) assigneeShouldReceiveSupplementalEmail(ctx context.Context, channelSet map[uint]struct{}, payload map[string]interface{}) bool {
	flags := s.channelKindFlagsForSet(ctx, channelSet)
	if flags.hasEmail && !flags.hasDingding && !flags.hasWeCom && !flags.hasWechat {
		return false
	}
	if flags.hasWechat && !flags.hasDingding && !flags.hasWeCom {
		return true
	}
	if flags.hasDingding || flags.hasWeCom {
		return s.anyAssigneePhoneNeedsMailFallback(ctx, channelSet, collectAssigneePhonesFromPayload(payload))
	}
	return true
}

// resolveAssigneeMailRecipients 规则处理人邮箱 + 值班邮箱；不含项目全员。
func (s *AlertService) resolveAssigneeMailRecipients(ctx context.Context, ruleID uint, status string) []string {
	var emails []string
	if s.assigneeSvc != nil && (status != "resolved" || s.assigneeSvc.NotifyOnResolvedEnabled(ctx, ruleID)) {
		e, _ := s.assigneeSvc.ResolveNotifyEmails(ctx, ruleID)
		emails = append(emails, e...)
	}
	if s.dutySvc != nil {
		e, _ := s.dutySvc.ResolveNotifyEmailsAtRule(ctx, ruleID, time.Now())
		emails = append(emails, e...)
	}
	return mergeNotifyEmailsUnique(emails)
}

// expandChannelSetForAssigneeNotification 命中接收组且规则有处理人时，按渠道策略补邮件。
func (s *AlertService) expandChannelSetForAssigneeNotification(
	ctx context.Context,
	channelSet map[uint]struct{},
	receiverGroupIDs []uint,
	payload map[string]interface{},
) {
	if s == nil || len(channelSet) == 0 || len(receiverGroupIDs) == 0 || payload == nil {
		return
	}
	status := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", payload["status"])))
	if status == "" {
		status = "firing"
	}
	if s.receiverGroupCache == nil {
		return
	}

	ruleID, hasRule := monitorRuleIDFromPayload(payload)
	if !hasRule || ruleID == 0 {
		return
	}
	mailTo := s.resolveAssigneeMailRecipients(ctx, ruleID, status)
	if len(mailTo) == 0 {
		return
	}
	payload["assignee_emails"] = mailTo

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
				channelSet[cid] = struct{}{}
			}
		}
	}
	if len(rgEmails) > 0 {
		payload["receiver_group_emails"] = mergeNotifyEmailsUnique(rgEmails)
	}

	needSupplement := s.assigneeShouldReceiveSupplementalEmail(ctx, channelSet, payload)
	if !needSupplement {
		return
	}

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
			payload["assignee_mail_fallback"] = true
		}
	}
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
