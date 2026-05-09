package service

import (
	"context"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
)

// channelNotifyFunc 单类 Webhook 通道投递（钉钉/企微/通用 Webhook）；邮件走独立 sendEmailChannel。
type channelNotifyFunc func(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}, settings map[string]interface{}) (int, string, error)

func (s *AlertService) buildNotifierRegistry() map[string]channelNotifyFunc {
	return map[string]channelNotifyFunc{
		alertdispatch.ChannelTypeGenericWebhook: s.notifyGenericWebhook,
		alertdispatch.ChannelTypeWechat:         s.notifyWeComWebhook,
		alertdispatch.ChannelTypeWechatWork:     s.notifyWeComWebhook,
		alertdispatch.ChannelTypeDingding:       s.notifyDingTalkWebhook,
	}
}
