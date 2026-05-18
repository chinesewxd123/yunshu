package dictconfig

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// MinioDictTypes MySQL 备份归档 MinIO 配置（数据字典权威来源，不写 config.yaml）。
type MinioDictTypes struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    string
	Region    string
	Prefix    string
}

func DefaultMinioDictTypes() MinioDictTypes {
	return MinioDictTypes{
		Endpoint:  "minio_endpoint",
		AccessKey: "minio_access_key",
		SecretKey: "minio_secret_key",
		Bucket:    "minio_bucket",
		UseSSL:    "minio_use_ssl",
		Region:    "minio_region",
		Prefix:    "minio_backup_prefix",
	}
}

// MinioConfig 运行时 MinIO 连接参数。
type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
	Prefix    string // 对象键前缀，如 mysql-backups/
}

func (c MinioConfig) Ready() bool {
	return strings.TrimSpace(c.Endpoint) != "" &&
		strings.TrimSpace(c.AccessKey) != "" &&
		strings.TrimSpace(c.SecretKey) != "" &&
		strings.TrimSpace(c.Bucket) != ""
}

// ResolveMinioConfig 从数据字典读取 MinIO 配置。
func ResolveMinioConfig(ctx context.Context, db *gorm.DB, types MinioDictTypes) MinioConfig {
	var cfg MinioConfig
	if db == nil {
		return cfg
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Endpoint); ok {
		cfg.Endpoint = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.AccessKey); ok {
		cfg.AccessKey = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.SecretKey); ok {
		cfg.SecretKey = strings.TrimSpace(v)
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Bucket); ok {
		cfg.Bucket = v
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.UseSSL); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.UseSSL = bv
		}
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Region); ok {
		cfg.Region = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.Prefix); ok {
		cfg.Prefix = strings.TrimSuffix(strings.TrimSpace(v), "/") + "/"
	} else {
		cfg.Prefix = "mysql-backups/"
	}
	return cfg
}
