package mailer

import (
	"context"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"

	"gorm.io/gorm"
)

// DictMailResolver 从数据字典 + YAML 底稿解析邮件配置。
type DictMailResolver struct {
	DB       *gorm.DB
	YAMLBase config.MailConfig
	Types    dictconfig.MailDictTypes
}

func (r *DictMailResolver) ResolveMailConfig(ctx context.Context) config.MailConfig {
	if r == nil {
		return config.MailConfig{}
	}
	types := r.Types
	if types.Host == "" {
		types = dictconfig.DefaultMailDictTypes()
	}
	return dictconfig.ResolveMailConfig(ctx, r.DB, r.YAMLBase, types)
}
