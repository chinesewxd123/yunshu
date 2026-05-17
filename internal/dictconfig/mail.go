package dictconfig

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

// MailDictTypes 数据字典中覆盖 mail.* 的 dict_type（与 bootstrap 一致）。
type MailDictTypes struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    string
}

func DefaultMailDictTypes() MailDictTypes {
	return MailDictTypes{
		Host:      "mail_host",
		Port:      "mail_port",
		Username:  "mail_username",
		Password:  "mail_password",
		FromEmail: "mail_from_email",
		FromName:  "mail_from_name",
		UseTLS:    "mail_use_tls",
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
	return strings.TrimSpace(row.Value), true
}

func fetchEnabledDictValueNonEmpty(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	v, ok := fetchEnabledDictValue(ctx, db, dictType)
	if !ok || strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

func parseBoolLoose(raw string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return false, false
	}
	switch s {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
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

// ResolveMailConfig 以 yamlBase 为底，用已启用的数据字典项覆盖（字典存在则优先）。
func ResolveMailConfig(ctx context.Context, db *gorm.DB, yamlBase config.MailConfig, types MailDictTypes) config.MailConfig {
	cfg := yamlBase
	if db == nil {
		return cfg
	}

	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Host); ok {
		cfg.Host = v
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.Port); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.Port = n
		}
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Username); ok {
		cfg.Username = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Password); ok {
		cfg.Password = strings.TrimSpace(v)
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.FromEmail); ok {
		cfg.FromEmail = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.FromName); ok {
		cfg.FromName = v
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.UseTLS); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.UseTLS = bv
		}
	}
	return cfg
}
