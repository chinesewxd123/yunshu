package bootstrap

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

type dictConfigOverrides struct {
	// Alert
	AlertWebhookTokenType    string
	AlertPrometheusURLType   string
	AlertPrometheusTokenType string

	// K8s Event Forward
	K8sEventForwardEnabledType               string
	K8sEventForwardWatcherBufferSizeType     string
	K8sEventForwardWorkerIntervalSecondsType string
	K8sEventForwardWorkerBatchSizeType       string
	K8sEventForwardWorkerMaxRetriesType      string

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

		K8sEventForwardEnabledType:               "k8s_event_forward_enabled",
		K8sEventForwardWatcherBufferSizeType:     "k8s_event_forward_watcher_buffer_size",
		K8sEventForwardWorkerIntervalSecondsType: "k8s_event_forward_worker_interval_seconds",
		K8sEventForwardWorkerBatchSizeType:       "k8s_event_forward_worker_batch_size",
		K8sEventForwardWorkerMaxRetriesType:      "k8s_event_forward_worker_max_retries",

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
		b.app.Logger.Biz("config").Infow(msg, kv...)
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

	// K8s Event Forward: 字典优先，YAML 兜底
	b.app.Config.K8sEventForward = dictconfig.ResolveK8sEventForwardConfig(
		ctx, b.app.DB, b.yamlK8sEventForwardBase, dictconfig.DefaultK8sEventForwardDictTypes(),
	)
	logf("k8s event forward config resolved (dict overrides yaml)",
		"enabled", b.app.Config.K8sEventForward.Enabled,
		"worker_interval_seconds", b.app.Config.K8sEventForward.WorkerIntervalSeconds,
	)

	// Mail: 字典优先，YAML 兜底（与发信时 DynamicSender 解析规则一致）
	types := dictconfig.MailDictTypes{
		Host:      ov.MailHostType,
		Port:      ov.MailPortType,
		Username:  ov.MailUsernameType,
		Password:  ov.MailPasswordType,
		FromEmail: ov.MailFromEmailType,
		FromName:  ov.MailFromNameType,
		UseTLS:    ov.MailUseTLSType,
	}
	b.app.Config.Mail = dictconfig.ResolveMailConfig(ctx, b.app.DB, b.yamlMailBase, types)
	if strings.TrimSpace(b.app.Config.Mail.Host) != "" {
		logf("mail config resolved (dict overrides yaml)",
			"host", b.app.Config.Mail.Host,
			"port", b.app.Config.Mail.Port,
			"from", b.app.Config.Mail.FromEmail,
		)
	}
}
