package config

import "github.com/spf13/viper"

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Log      LogConfig      `mapstructure:"log"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Mail     MailConfig     `mapstructure:"mail"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Casbin   CasbinConfig   `mapstructure:"casbin"`
	Swagger  SwaggerConfig  `mapstructure:"swagger"`
	Alert    AlertConfig    `mapstructure:"alert"`
	Security SecurityConfig `mapstructure:"security"`
	Agent    AgentConfig    `mapstructure:"agent"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

type HTTPConfig struct {
	ReadHeaderTimeoutSeconds int `mapstructure:"read_header_timeout_seconds"`
	ReadTimeoutSeconds       int `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds      int `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds       int `mapstructure:"idle_timeout_seconds"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`    // console, file, both
	FilePath string `mapstructure:"file_path"` // log file directory path
}

type MySQLConfig struct {
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	DBName                 string `mapstructure:"db_name"`
	Charset                string `mapstructure:"charset"`
	Loc                    string `mapstructure:"loc"`
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`
	ConnMaxLifetimeSeconds int    `mapstructure:"conn_max_lifetime_seconds"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type MailConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromEmail string `mapstructure:"from_email"`
	FromName  string `mapstructure:"from_name"`
	UseTLS    bool   `mapstructure:"use_tls"`
}

type AuthConfig struct {
	JWTSecret                string `mapstructure:"jwt_secret"`
	AccessTokenTTLMinutes    int    `mapstructure:"access_token_ttl_minutes"`
	EmailCodeTTLSeconds      int    `mapstructure:"email_code_ttl_seconds"`
	EmailCodeCooldownSeconds int    `mapstructure:"email_code_cooldown_seconds"`
}

type CasbinConfig struct {
	ModelPath string `mapstructure:"model_path"`
}

type SwaggerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type SecurityConfig struct {
	// EncryptionKey is used to encrypt sensitive data (e.g., server SSH credentials).
	// Recommended: 32-byte key in base64 format.
	EncryptionKey string `mapstructure:"encryption_key"`
}

type AgentConfig struct {
	// RegisterSecret is a shared secret used by log-agent to register itself and obtain an ingest token.
	// If empty, public registration endpoint is disabled.
	RegisterSecret string `mapstructure:"register_secret"`
}

type AlertConfig struct {
	WebhookToken     string `mapstructure:"webhook_token"`
	DefaultTimeoutMS int    `mapstructure:"default_timeout_ms"`
	MaxPayloadChars  int    `mapstructure:"max_payload_chars"`
	DedupTTLSeconds  int    `mapstructure:"dedup_ttl_seconds"`
	PrometheusURL    string `mapstructure:"prometheus_url"`
	PrometheusToken  string `mapstructure:"prometheus_token"`
	PromQueryTimeout int    `mapstructure:"prom_query_timeout_seconds"`
	// P3 异步增强：通知主链路不阻塞 Prometheus 查询。
	PrometheusEnrichEnabled   bool `mapstructure:"prometheus_enrich_enabled"`
	PrometheusEnrichQueueSize int  `mapstructure:"prometheus_enrich_queue_size"`
	PrometheusEnrichWorkers   int  `mapstructure:"prometheus_enrich_workers"`

	// GroupBy 决定 group_key 计算维度，用于服务端聚合/收敛。
	// 建议包含：alertname, cluster, namespace, severity, receiver
	GroupBy []string `mapstructure:"group_by"`
	// NotifyIntervalSeconds: 同一个 group_key 在 firing 状态下的最小发送间隔（秒）。
	NotifyIntervalSeconds int `mapstructure:"notify_interval_seconds"`
	// ResolvedNotifyIntervalSeconds: 同一个 group_key 在 resolved 状态下的最小发送间隔（秒），用于恢复汇总去抖。
	ResolvedNotifyIntervalSeconds int `mapstructure:"resolved_notify_interval_seconds"`
	// AggregateTTLSeconds: group_key 状态在 Redis 中的过期时间（秒）。
	AggregateTTLSeconds int `mapstructure:"aggregate_ttl_seconds"`

	PlatformLimits AlertPlatformLimits `mapstructure:"platform_limits"`
}

type AlertPlatformLimits struct {
	DingdingMaxChars int `mapstructure:"dingding_max_chars"`
	WeComMaxChars    int `mapstructure:"wecom_max_chars"`
	GenericMaxChars  int `mapstructure:"generic_max_chars"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.Alert.DefaultTimeoutMS <= 0 {
		cfg.Alert.DefaultTimeoutMS = 5000
	}
	if cfg.Alert.MaxPayloadChars <= 0 {
		cfg.Alert.MaxPayloadChars = 8000
	}
	if cfg.Alert.DedupTTLSeconds <= 0 {
		cfg.Alert.DedupTTLSeconds = 86400
	}
	if cfg.Alert.PromQueryTimeout <= 0 {
		cfg.Alert.PromQueryTimeout = 5
	}
	if cfg.Alert.PrometheusEnrichQueueSize <= 0 {
		cfg.Alert.PrometheusEnrichQueueSize = 1024
	}
	if cfg.Alert.PrometheusEnrichWorkers <= 0 {
		cfg.Alert.PrometheusEnrichWorkers = 4
	}
	if cfg.Alert.NotifyIntervalSeconds <= 0 {
		cfg.Alert.NotifyIntervalSeconds = 300
	}
	if cfg.Alert.ResolvedNotifyIntervalSeconds <= 0 {
		cfg.Alert.ResolvedNotifyIntervalSeconds = 30
	}
	if cfg.Alert.AggregateTTLSeconds <= 0 {
		cfg.Alert.AggregateTTLSeconds = 86400
	}
	if len(cfg.Alert.GroupBy) == 0 {
		cfg.Alert.GroupBy = []string{"alertname", "cluster", "namespace", "severity", "receiver"}
	}
	// 平台长度限制：预留空间给 @ 和格式控制，默认值偏保守。
	if cfg.Alert.PlatformLimits.DingdingMaxChars <= 0 {
		cfg.Alert.PlatformLimits.DingdingMaxChars = 4500
	}
	if cfg.Alert.PlatformLimits.WeComMaxChars <= 0 {
		cfg.Alert.PlatformLimits.WeComMaxChars = 3500
	}
	if cfg.Alert.PlatformLimits.GenericMaxChars <= 0 {
		cfg.Alert.PlatformLimits.GenericMaxChars = 8000
	}
	return &cfg, nil
}
