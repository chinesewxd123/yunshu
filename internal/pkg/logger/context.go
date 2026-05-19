package logger

import (
	"context"
	"strconv"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota + 1
	ctxKeyUserID
	ctxKeyUsername
)

// ContextExtractors 从 context 提取日志字段（对齐 onex log.WithContextExtractor）。
type ContextExtractors map[string]func(context.Context) string

var contextExtractors = ContextExtractors{
	"request_id": extractRequestID,
	"user_id":    extractUserID,
	"username":   extractUsername,
}

// RegisterContextExtractor 注册自定义 context 字段提取器。
func RegisterContextExtractor(name string, fn func(context.Context) string) {
	if name == "" || fn == nil {
		return
	}
	contextExtractors[name] = fn
}

// RegisterContextExtractors 批量注册提取器。
func RegisterContextExtractors(ext ContextExtractors) {
	for k, v := range ext {
		RegisterContextExtractor(k, v)
	}
}

// WithRequestID 将 request_id 写入标准 context（HTTP 中间件在请求入口调用）。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil || requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

// RequestIDFromContext 读取 request_id。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// WithUser 将当前用户写入 context（鉴权中间件调用）。
func WithUser(ctx context.Context, userID uint, username string) context.Context {
	if ctx == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	if username != "" {
		ctx = context.WithValue(ctx, ctxKeyUsername, username)
	}
	return ctx
}

func extractRequestID(ctx context.Context) string {
	return RequestIDFromContext(ctx)
}

func extractUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, ok := ctx.Value(ctxKeyUserID).(uint)
	if !ok || id == 0 {
		return ""
	}
	return formatUint(id)
}

func extractUsername(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyUsername).(string)
	return v
}

// attrsFromContext 按注册的提取器生成 key-value 对。
func attrsFromContext(ctx context.Context) []any {
	if ctx == nil || len(contextExtractors) == 0 {
		return nil
	}
	var out []any
	for name, fn := range contextExtractors {
		if fn == nil {
			continue
		}
		if val := fn(ctx); val != "" {
			out = append(out, name, val)
		}
	}
	return out
}

func formatUint(v uint) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(v), 10)
}
