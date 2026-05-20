package logger

import (
	"context"
	"errors"
	"net/http"

	"yunshu/internal/pkg/apperror"
)

var defaultLogger *Logger

// SetDefault 进程启动时注入全局 Logger。
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Default 返回全局 Logger。
func Default() *Logger {
	return defaultLogger
}

// Biz 返回默认 Logger 上的业务组件（layer=service）。
func Biz(component string) *Component {
	return &Component{component: component, layer: LayerService, log: defaultLogger}
}

// W 从 context 提取 request_id、user 等字段（对齐 onex log.W）；需再 WithLayer / 配合 Biz 使用 component。
func W(ctx context.Context) *Component {
	return &Component{log: defaultLogger, ctx: ctx}
}

// Component 业务组件日志：自动写入 layer、component，并按级别分流到 info/error 文件。
// 通过 W(ctx) 注入 request_id / user 等字段（对齐 onex log.W）。
type Component struct {
	component string
	layer     string
	log       *Logger
	ctx       context.Context
}

// WithLayer 复制并指定分层（http/api/service/dao/grpc/worker）。
func (b *Component) WithLayer(layer string) *Component {
	if b == nil {
		return nil
	}
	cp := *b
	cp.layer = layer
	return &cp
}

// W 复制组件并绑定 context，后续 Info/Warn/Error 自动带上提取字段。
func (b *Component) W(ctx context.Context) *Component {
	if b == nil {
		return nil
	}
	cp := *b
	cp.ctx = ctx
	return &cp
}

func (b *Component) enabled() bool {
	return b != nil && b.log != nil
}

func (b *Component) baseAttrs(attrs []any) []any {
	capHint := len(attrs) + 8
	if b.ctx != nil {
		capHint += len(contextExtractors) * 2
	}
	out := make([]any, 0, capHint)
	if b.ctx != nil {
		out = append(out, attrsFromContext(b.ctx)...)
	}
	if b.layer != "" {
		out = append(out, "layer", b.layer)
	}
	if b.component != "" {
		out = append(out, "component", b.component)
	}
	return append(out, attrs...)
}

// Info 写入 info.log。
func (b *Component) Info(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Info.Info(msg, b.baseAttrs(attrs)...)
}

// Warn 写入 info.log（与 Info 同文件）。
func (b *Component) Warn(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Info.Warn(msg, b.baseAttrs(attrs)...)
}

// Error 写入 error.log。
func (b *Component) Error(msg string, attrs ...any) {
	if !b.enabled() {
		return
	}
	b.log.Error.Error(msg, b.baseAttrs(attrs)...)
}

// Op 记录操作失败：5xx→Error 文件，4xx→Warn（info 文件），其它→Error。
func (b *Component) Op(operation string, err error, attrs ...any) {
	if err == nil {
		return
	}
	attrs = append(attrs, "operation", operation, "error", err)
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs = append(attrs, "error_code", appErr.ErrorCode, "reason", appErr.Reason, "http_status", appErr.StatusCode)
		if appErr.StatusCode >= http.StatusInternalServerError {
			b.Error("Operation failed", attrs...)
			return
		}
		if appErr.StatusCode >= http.StatusBadRequest {
			b.Warnw("Operation rejected", attrs...)
			return
		}
	}
	b.Error("Operation failed", attrs...)
}

// LogErr 包级快捷：组件操作失败。
func LogErr(component, operation string, err error, attrs ...any) {
	if err == nil {
		return
	}
	Biz(component).Op(operation, err, attrs...)
}

// LogErrCtx 带 context 的 LogErr（自动附加 request_id / user）。
func LogErrCtx(ctx context.Context, component, operation string, err error, attrs ...any) {
	if err == nil {
		return
	}
	Biz(component).W(ctx).Op(operation, err, attrs...)
}

// Infow / Warnw / Errorw 对齐 onex 结构化写法（keyvals 成对出现）。
func (b *Component) Infow(msg string, keyvals ...any)  { b.Info(msg, keyvals...) }
func (b *Component) Warnw(msg string, keyvals ...any)  { b.Warn(msg, keyvals...) }
func (b *Component) Errorw(err error, msg string, keyvals ...any) {
	if err != nil {
		keyvals = append(keyvals, "error", err)
	}
	b.Error(msg, keyvals...)
}

func Infow(msg string, keyvals ...any)            { Biz("app").Infow(msg, keyvals...) }
func Warnw(msg string, keyvals ...any)            { Biz("app").Warnw(msg, keyvals...) }
func Errorw(err error, msg string, keyvals ...any) { Biz("app").Errorw(err, msg, keyvals...) }
