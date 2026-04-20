package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/middleware"
	"yunshu/internal/model"
	"yunshu/internal/pkg/casbinadapter"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/mailer"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// App 运行时全局依赖：配置、日志、DB、Redis、Casbin、邮件与 Gin 引擎。
type App struct {
	Config   *config.Config
	Logger   *logx.Logger
	DB       *gorm.DB
	Redis    *redis.Client
	Enforcer *casbin.SyncedEnforcer
	Mailer   mailer.Sender
	Engine   *gin.Engine
}

type Builder struct {
	app *App
	err error
}

func NewBuilder() *Builder {
	return &Builder{app: &App{}}
}

func (b *Builder) WithConfig(path string) *Builder {
	if b.err != nil {
		return b
	}

	cfg, err := config.Load(path)
	if err != nil {
		b.err = err
		return b
	}
	b.app.Config = cfg
	return b
}

func (b *Builder) WithLogger() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before logger")
		return b
	}

	b.app.Logger = logx.New(b.app.Config.Log)
	return b
}

func (b *Builder) WithMySQL() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before mysql")
		return b
	}

	cfg := b.app.Config.MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.Charset,
		cfg.Loc,
	)

	// gormLogLevel := gormlogger.Silent
	// if b.app.Config.Log.Level == "debug" {
	// 	gormLogLevel = gormlogger.Info
	// }

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logx.NewGormLogger(b.app.Logger.SQL, b.app.Config.Log.Level),
	})
	if err != nil {
		b.err = err
		return b
	}
	// 自定义关联表	user_roles,可以在自定义表中添加额外的字段、自定义索引等
	if err = db.SetupJoinTable(&model.User{}, "Roles", &model.UserRole{}); err != nil {
		b.err = err
		return b
	}

	sqlDB, err := db.DB()
	if err != nil {
		b.err = err
		return b
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSeconds) * time.Second)

	b.app.DB = db
	return b
}

// WithDictOverrides 在 MySQL 已就绪后，从数据字典覆盖“运行期可变”的配置项（告警域 + 邮件）。
// 注意：mysql/redis/app.env/grpc.listen_addr 等启动期项仍以 env/yaml 为准，避免启动鸡生蛋。
func (b *Builder) WithDictOverrides() *Builder {
	if b.err != nil {
		return b
	}
	if b.app == nil || b.app.Config == nil || b.app.DB == nil {
		// MySQL 未就绪则跳过；不作为错误
		return b
	}
	b.applyDictConfigOverrides(context.Background(), defaultDictConfigOverrides())
	return b
}

func (b *Builder) WithRedis() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before redis")
		return b
	}

	cfg := b.app.Config.Redis
	b.app.Redis = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	return b
}

func (b *Builder) WithCasbin() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil || b.app.DB == nil {
		b.err = errors.New("config and mysql are required before casbin")
		return b
	}

	// Defensive cleanup: if casbin_rule contains malformed rows (e.g. empty ptype),
	// casbin's LoadPolicyLine may panic. Keep startup resilient by pruning obviously
	// invalid rows before adapter loads policies.
	// Keep Casbin startup resilient: prune malformed rules that can make casbin panic
	// when parsing policy lines (e.g. invalid/garbage ptype).
	//
	// Valid Casbin ptype is typically: p, g, p2, g2, ...
	_ = b.app.DB.Exec("DELETE FROM casbin_rule WHERE ptype IS NULL OR ptype = '' OR ptype NOT REGEXP '^(p|g)[0-9]*$'").Error

	adapter := casbinadapter.NewSafeGormAdapter(b.app.DB, "casbin_rule")

	enforcer, err := casbin.NewSyncedEnforcer(b.app.Config.Casbin.ModelPath, adapter)
	if err != nil {
		b.err = err
		return b
	}
	if err = enforcer.LoadPolicy(); err != nil {
		b.err = err
		return b
	}

	b.app.Enforcer = enforcer
	return b
}

func (b *Builder) WithMailer() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before mailer")
		return b
	}

	b.app.Mailer = mailer.NewSMTPSender(b.app.Config.Mail)
	return b
}

func (b *Builder) WithGin() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil || b.app.Logger == nil {
		b.err = errors.New("config and logger are required before gin")
		return b
	}

	if b.app.Config.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.Recovery(b.app.Logger))
	engine.Use(middleware.RequestLogger(b.app.Logger))
	b.app.Engine = engine
	return b
}

func (b *Builder) Build() (*App, error) {
	return b.app, b.err
}

func (a *App) Close() error {
	var errs []error

	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				errs = append(errs, closeErr)
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
