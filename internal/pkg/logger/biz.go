package logger

import (
	"errors"
	"log/slog"
	"net/http"

	"yunshu/internal/pkg/apperror"
)

var defaultLogger *Logger

// SetDefault 设置业务层默认日志（在进程启动早期调用一次）。
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Default 返回进程级默认 Logger，未设置时为 nil。
func Default() *Logger {
	return defaultLogger
}

// Biz 返回带组件名的业务日志器，复用 Info / Error 通道。
func Biz(component string) *Component {
	return &Component{component: component, log: defaultLogger}
}

// Component 业务组件日志，不重复实现 slog 配置。
type Component struct {
	component string
	log       *Logger
}

func (b *Component) enabled() bool {
	return b != nil && b.log != nil
}

func (b *Component) withComponent(attrs []any) []any {
	out := make([]any, 0, len(attrs)+2)
	out = append(out, "component", b.component)
	return append(out, attrs...)
}

// Info 记录 info 级业务事件。
func (b *Component) Info(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Info.Info(msg, b.withComponent(attrs)...)
}

// Warn 记录 warn 级业务事件。
func (b *Component) Warn(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Info.Warn(msg, b.withComponent(attrs)...)
}

// Debug 记录 debug 级业务事件。
func (b *Component) Debug(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Info.Debug(msg, b.withComponent(attrs)...)
}

// Error 记录 error 级业务事件。
func (b *Component) Error(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Error.Error(msg, b.withComponent(attrs)...)
}

// Op 记录操作失败；自动附加 error 与 operation。
func (b *Component) Op(operation string, err error, attrs ...any) {
	if err == nil {
		return
	}
	attrs = append(attrs, "operation", operation, slog.Any("error", err))
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs = append(attrs, "error_code", appErr.ErrorCode, "http_status", appErr.StatusCode)
		if appErr.StatusCode >= http.StatusInternalServerError {
			b.Error("operation failed", attrs...)
			return
		}
		if appErr.StatusCode >= http.StatusBadRequest {
			b.Debug("operation rejected", attrs...)
			return
		}
	}
	b.Error("operation failed", attrs...)
}

// LogErr 在 default Logger 上记录组件操作失败（包级快捷函数）。
func LogErr(component, operation string, err error, attrs ...any) {
	if err == nil {
		return
	}
	Biz(component).Op(operation, err, attrs...)
}
