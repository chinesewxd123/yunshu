package alertdispatch

import "strings"

// 与 alert_channels.type 及历史行为保持一致，供注册表键与规范化使用。
const (
	ChannelTypeEmail          = "email"
	ChannelTypeGenericWebhook = "generic_webhook"
	ChannelTypeWechat         = "wechat"
	ChannelTypeWechatWork     = "wechat_work"
	ChannelTypeDingding       = "dingding"
)

// NormalizeWebhookChannelType 将通道类型规范为注册表键；空值回退为通用 Webhook（与既有逻辑一致）。
func NormalizeWebhookChannelType(t string) string {
	v := strings.ToLower(strings.TrimSpace(t))
	if v == "" {
		return ChannelTypeGenericWebhook
	}
	return v
}
