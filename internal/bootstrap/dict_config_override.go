package bootstrap

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type dictConfigOverrides struct {
	// Alert
	AlertWebhookTokenType    string
	AlertPrometheusURLType   string
	AlertPrometheusTokenType string

	// Mail
	MailHostType      string
	MailPortType      string
	MailUsernameType  string
	MailPasswordType  string
	MailFromEmailType string
	MailFromNameType  string
	MailUseTLSType    string
}

func defaultDictConfigOverrides() dictConfigOverrides {
	return dictConfigOverrides{
		AlertWebhookTokenType:    "alert_webhook_token",
		AlertPrometheusURLType:   "alert_enrich_prometheus_url",
		AlertPrometheusTokenType: "alert_enrich_prometheus_token",

		MailHostType:      "mail_host",
		MailPortType:      "mail_port",
		MailUsernameType:  "mail_username",
		MailPasswordType:  "mail_password",
		MailFromEmailType: "mail_from_email",
		MailFromNameType:  "mail_from_name",
		MailUseTLSType:    "mail_use_tls",
	}
}

func fetchEnabledDictValue(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	if db == nil || strings.TrimSpace(dictType) == "" {
		return "", false
	}
	var row model.DictEntry
	err := db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND status = 1", strings.TrimSpace(dictType)).
		Order("sort ASC, id DESC").
		Limit(1).
		First(&row).Error
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(row.Value)
	// 允许空字符串作为显式覆盖（例如 token 置空），由调用方决定是否接受
	return v, true
}

func fetchEnabledDictValueNonEmpty(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	v, ok := fetchEnabledDictValue(ctx, db, dictType)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

func parseBoolLoose(raw string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return false, false
	}
	if s == "1" || s == "true" || s == "yes" || s == "y" || s == "on" {
		return true, true
	}
	if s == "0" || s == "false" || s == "no" || s == "n" || s == "off" {
		return false, true
	}
	return false, false
}

func parseInt(raw string) (int, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// applyDictConfigOverrides best-effort 覆盖 app.Config 中的 alert/mail 配置项。
// 约束：MySQL 连接已建立（可读 dict_entries 表）。
func (b *Builder) applyDictConfigOverrides(ctx context.Context, ov dictConfigOverrides) {
	if b == nil || b.app == nil || b.app.Config == nil || b.app.DB == nil {
		return
	}

	logf := func(msg string, kv ...any) {
		if b.app.Logger == nil {
			return
		}
		// 用 Info logger，避免引入新的 logger API 依赖；具体格式由 logger 实现决定
		b.app.Logger.Info.Info(msg, kv...)
	}

	// Alert: webhook_token
	if v, ok := fetchEnabledDictValue(ctx, b.app.DB, ov.AlertWebhookTokenType); ok {
		b.app.Config.Alert.WebhookToken = v
		logf("config override from dict", "key", "alert.webhook_token", "dict_type", ov.AlertWebhookTokenType)
	}
	// Alert: prometheus_url
	if v, ok := fetchEnabledDictValue(ctx, b.app.DB, ov.AlertPrometheusURLType); ok {
		b.app.Config.Alert.PrometheusURL = v
		logf("config override from dict", "key", "alert.prometheus_url", "dict_type", ov.AlertPrometheusURLType)
	}
	// Alert: prometheus_token (sensitive) - allow empty string override
	if v, ok := fetchEnabledDictValue(ctx, b.app.DB, ov.AlertPrometheusTokenType); ok {
		b.app.Config.Alert.PrometheusToken = v
		logf("config override from dict", "key", "alert.prometheus_token", "dict_type", ov.AlertPrometheusTokenType, "sensitive", true)
	}

	// Mail: host（仅允许非空值覆盖，避免空值冲掉 YAML）
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, b.app.DB, ov.MailHostType); ok {
		b.app.Config.Mail.Host = v
		logf("config override from dict", "key", "mail.host", "dict_type", ov.MailHostType)
	}
	// Mail: port
	if v, ok := fetchEnabledDictValue(ctx, b.app.DB, ov.MailPortType); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			b.app.Config.Mail.Port = n
			logf("config override from dict", "key", "mail.port", "dict_type", ov.MailPortType)
		} else {
			logf("config override skipped (invalid int)", "key", "mail.port", "dict_type", ov.MailPortType)
		}
	}
	// Mail: username
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, b.app.DB, ov.MailUsernameType); ok {
		b.app.Config.Mail.Username = v
		logf("config override from dict", "key", "mail.username", "dict_type", ov.MailUsernameType)
	}
	// Mail: password (sensitive) - 禁止空字符串覆盖，避免 SMTP 认证失败
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, b.app.DB, ov.MailPasswordType); ok {
		b.app.Config.Mail.Password = v
		logf("config override from dict", "key", "mail.password", "dict_type", ov.MailPasswordType, "sensitive", true)
	}
	// Mail: from_email
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, b.app.DB, ov.MailFromEmailType); ok {
		b.app.Config.Mail.FromEmail = v
		logf("config override from dict", "key", "mail.from_email", "dict_type", ov.MailFromEmailType)
	}
	// Mail: from_name
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, b.app.DB, ov.MailFromNameType); ok {
		b.app.Config.Mail.FromName = v
		logf("config override from dict", "key", "mail.from_name", "dict_type", ov.MailFromNameType)
	}
	// Mail: use_tls
	if v, ok := fetchEnabledDictValue(ctx, b.app.DB, ov.MailUseTLSType); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			b.app.Config.Mail.UseTLS = bv
			logf("config override from dict", "key", "mail.use_tls", "dict_type", ov.MailUseTLSType)
		} else {
			logf("config override skipped (invalid bool)", "key", "mail.use_tls", "dict_type", ov.MailUseTLSType)
		}
	}
}
