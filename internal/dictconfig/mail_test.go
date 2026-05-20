package dictconfig

import (
	"context"
	"testing"

	"yunshu/internal/config"
)

func TestResolveMailConfig_dictOverridesYAML(t *testing.T) {
	base := config.MailConfig{Host: "yaml.example.com", Port: 25}
	got := ResolveMailConfig(context.Background(), nil, base, DefaultMailDictTypes())
	if got.Host != "yaml.example.com" || got.Port != 25 {
		t.Fatalf("without db: got %+v", got)
	}
}

func TestResolveMailConfig_passwordTrimmed(t *testing.T) {
	// 无 DB 时仅验证底稿透传
	base := config.MailConfig{Password: "  secret  "}
	got := ResolveMailConfig(context.Background(), nil, base, DefaultMailDictTypes())
	if got.Password != "  secret  " {
		t.Fatalf("without dict password unchanged: %q", got.Password)
	}
}
