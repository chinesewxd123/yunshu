package config

import "github.com/spf13/viper"

type Config struct {
	App     AppConfig     `mapstructure:"app"`
	HTTP    HTTPConfig    `mapstructure:"http"`
	Log     LogConfig     `mapstructure:"log"`
	MySQL   MySQLConfig   `mapstructure:"mysql"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Mail    MailConfig    `mapstructure:"mail"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Casbin  CasbinConfig  `mapstructure:"casbin"`
	Swagger SwaggerConfig `mapstructure:"swagger"`
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
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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
	return &cfg, nil
}
