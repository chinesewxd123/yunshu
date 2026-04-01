// internal/pkg/logger/gorm_logger.go
package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

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
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.SQLLogger.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.SQLLogger.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.SQLLogger.Error(fmt.Sprintf(msg, data...))
	}
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.SQLLogger.Error("SQL error",
			"error", err,
			"duration", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
		return
	}

	if l.SlowThreshold > 0 && elapsed > l.SlowThreshold {
		l.SQLLogger.Warn("Slow SQL",
			"duration", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
		return
	}

	if l.LogLevel == gormlogger.Info {
		l.SQLLogger.Info("SQL query",
			"duration", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)
	}
}
