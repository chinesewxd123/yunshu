package bootstrap

import (
	"errors"
	"fmt"
	"time"

	"go-permission-system/internal/config"
	"go-permission-system/internal/middleware"
	"go-permission-system/internal/model"
	logx "go-permission-system/internal/pkg/logger"
	"go-permission-system/internal/pkg/mailer"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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

	adapter, err := gormadapter.NewAdapterByDB(b.app.DB)
	if err != nil {
		b.err = err
		return b
	}

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
