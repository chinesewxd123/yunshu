package mailer

import (
	"context"

	"yunshu/internal/config"
)

// MailConfigResolver 在每次发信前解析 SMTP 配置（用于数据字典运行期覆盖）。
type MailConfigResolver interface {
	ResolveMailConfig(ctx context.Context) config.MailConfig
}

// DynamicSender 按次解析配置后发信，保证数据字典修改后无需重启进程。
type DynamicSender struct {
	resolver MailConfigResolver
}

func NewDynamicSender(resolver MailConfigResolver) *DynamicSender {
	return &DynamicSender{resolver: resolver}
}

func (s *DynamicSender) resolve(ctx context.Context) config.MailConfig {
	if s == nil || s.resolver == nil {
		return config.MailConfig{}
	}
	return s.resolver.ResolveMailConfig(ctx)
}

func (s *DynamicSender) Enabled() bool {
	return NewSMTPSender(s.resolve(context.Background())).Enabled()
}

func (s *DynamicSender) Send(ctx context.Context, toEmail, subject, textBody string) error {
	return NewSMTPSender(s.resolve(ctx)).Send(ctx, toEmail, subject, textBody)
}

func (s *DynamicSender) SendMultipart(ctx context.Context, toEmail, subject, textPlain, htmlBody string) error {
	return NewSMTPSender(s.resolve(ctx)).SendMultipart(ctx, toEmail, subject, textPlain, htmlBody)
}
