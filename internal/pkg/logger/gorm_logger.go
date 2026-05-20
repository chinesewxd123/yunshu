package logger

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger 将 GORM SQL 写入 sql.log（结构化字段，不使用 fmt 拼消息）。
type GormLogger struct {
	SQLLogger     *slog.Logger
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}

func NewGormLogger(sqlLogger *slog.Logger, level string) *GormLogger {
	logLevel := gormlogger.Warn
	if level == "debug" {
		logLevel = gormlogger.Info
	}
	return &GormLogger{
		SQLLogger:     sqlLogger,
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      logLevel,
	}
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	cp := *l
	cp.LogLevel = level
	return &cp
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Info || l.SQLLogger == nil {
		return
	}
	l.SQLLogger.Info("gorm info", l.dataAttrs(msg, data)...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Warn || l.SQLLogger == nil {
		return
	}
	l.SQLLogger.Warn("gorm warn", l.dataAttrs(msg, data)...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel < gormlogger.Error || l.SQLLogger == nil {
		return
	}
	l.SQLLogger.Error("gorm error", l.dataAttrs(msg, data)...)
}

func (l *GormLogger) dataAttrs(msg string, data []interface{}) []any {
	attrs := []any{
		slog.String("layer", LayerDAO),
		slog.String("component", "gorm"),
		slog.String("message", msg),
	}
	for i, v := range data {
		attrs = append(attrs, slog.Any("arg_"+strconv.Itoa(i), v))
	}
	return attrs
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent || l.SQLLogger == nil {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	attrs := []any{
		slog.String("layer", LayerDAO),
		slog.String("component", "gorm"),
		slog.Duration("duration", elapsed),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	}
	if ctx != nil {
		attrs = append(attrs, attrsFromContext(ctx)...)
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.SQLLogger.Error("sql error", append(attrs, "error", err)...)
		return
	}

	if l.SlowThreshold > 0 && elapsed > l.SlowThreshold {
		l.SQLLogger.Warn("slow sql", attrs...)
		return
	}

	if l.LogLevel >= gormlogger.Info {
		l.SQLLogger.Info("sql query", attrs...)
	}
}
