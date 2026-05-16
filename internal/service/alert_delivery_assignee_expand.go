package service

import (
	"context"
	"strings"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
)

// expandChannelSetForAssigneeNotification 规则处理人/值班有邮箱且命中接收组时，补启邮件通道投递至 assignee_emails。
// 场景：接收组仅绑钉钉、处理人不在群内时，仍向监控规则「处理人」邮箱发信（不依赖接收组 email_recipients）。
func (s *AlertService) expandChannelSetForAssigneeNotification(
	ctx context.Context,
	channelSet map[uint]struct{},
	receiverGroupIDs []uint,
	payload map[string]interface{},
) {
	if s == nil || len(channelSet) == 0 || len(receiverGroupIDs) == 0 || payload == nil {
		return
	}
	assignee := collectEmailsFromPayload(payload, "assignee_emails")
	if len(assignee) == 0 {
		return
	}
	if s.receiverGroupCache == nil {
		return
	}

	var rgEmails []string
	hasEmailChannelInGroups := false
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
	merged := mergeNotifyEmailsUnique(append(assignee, rgEmails...))
	if len(merged) > 0 {
		payload["assignee_emails"] = merged
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
		}
	}
	_ = hasEmailChannelInGroups // 接收组是否含邮件通道仅影响是否已选入 channelSet，不阻断处理人邮件兜底
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
