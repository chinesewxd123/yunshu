package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: yaml-app
  env: prod
  port: 8080
grpc:
  listen_addr: 127.0.0.1:18080
  target_addr: 127.0.0.1:18080
mysql:
  host: mysql-yaml
  port: 3306
  user: root
  password: yaml-pass
  db_name: yaml-db
  charset: utf8mb4
  loc: Asia%2FShanghai
redis:
  addr: redis-yaml:6379
  password: yaml-redis
  db: 0
auth:
  jwt_secret: yaml-jwt
security:
  encryption_key: yaml-encryption
alert:
  default_timeout_ms: 5000
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	t.Setenv("APP_ENV", "dev")
	t.Setenv("MYSQL_HOST", "mysql-env")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_USER", "env-user")
	t.Setenv("MYSQL_PASSWORD", "env-pass")
	t.Setenv("MYSQL_DB_NAME", "env-db")
	t.Setenv("REDIS_ADDR", "redis-env:6380")
	t.Setenv("REDIS_PASSWORD", "env-redis")
	t.Setenv("JWT_SECRET", "env-jwt")
	t.Setenv("ENCRYPTION_KEY", "env-encryption")
	t.Setenv("ALERT_DEFAULT_TIMEOUT_MS", "7000")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Env != "dev" {
		t.Fatalf("expected APP_ENV override, got %q", cfg.App.Env)
	}
	if cfg.MySQL.Host != "mysql-env" || cfg.MySQL.Port != 3307 || cfg.MySQL.User != "env-user" || cfg.MySQL.Password != "env-pass" || cfg.MySQL.DBName != "env-db" {
		t.Fatalf("mysql env overrides not applied: %+v", cfg.MySQL)
	}
	if cfg.Redis.Addr != "redis-env:6380" || cfg.Redis.Password != "env-redis" {
		t.Fatalf("redis env overrides not applied: %+v", cfg.Redis)
	}
	if cfg.Auth.JWTSecret != "env-jwt" {
		t.Fatalf("expected JWT_SECRET alias override, got %q", cfg.Auth.JWTSecret)
	}
	if cfg.Auth.AccessTokenTTLMinutes != 120 || cfg.Auth.EmailCodeTTLSeconds != 600 || cfg.Auth.EmailCodeCooldownSeconds != 60 {
		t.Fatalf("auth defaults not applied: %+v", cfg.Auth)
	}
	if cfg.Security.EncryptionKey != "env-encryption" {
		t.Fatalf("expected ENCRYPTION_KEY alias override, got %q", cfg.Security.EncryptionKey)
	}
	if cfg.Alert.DefaultTimeoutMS != 7000 {
		t.Fatalf("expected ALERT_DEFAULT_TIMEOUT_MS override, got %d", cfg.Alert.DefaultTimeoutMS)
	}
}
