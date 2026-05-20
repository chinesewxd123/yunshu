// Package svcerr 在返回业务错误前写入统一业务日志（复用 logger.Biz / W(ctx)）。
//
// 分层约定（运维日志，非业务审计；审计走 login_logs / operation_logs 等表）：
//   - Service：svcerr.Pass(ctx, ...) / Internal / Warn / Reject — 失败、异常、可疑请求
//   - HTTP：response.logHTTPError — 仅补充 Service 未记过的错误（apperror.AlreadyLogged）
//   - 访问：RequestLogger — 每条 HTTP 一行（request_id / status / latency）
package svcerr

import (
	"context"
	"errors"
	"fmt"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service/svclog"
)

func biz(component string) *logx.Component {
	return svclog.Service(component)
}

// Internal 记录内部错误并返回 10901 AppError（msgFmt 为 constants.ErrFmt*，须含一个 %v 占位给 err）。
func Internal(ctx context.Context, component, operation string, err error, msgFmt string, attrs ...any) error {
	if err == nil {
		return nil
	}
	biz(component).W(ctx).Op(operation, err, attrs...)
	return apperror.MarkLogged(constants.ErrInternalWithMsg(fmt.Sprintf(msgFmt, err)))
}

// Internalf 与 Internal 相同，但 msgFmt 可含多个 fmt 占位（最后一个参数须为 err）。
func Internalf(ctx context.Context, component, operation string, err error, msgFmt string, args ...any) error {
	if err == nil {
		return nil
	}
	biz(component).W(ctx).Op(operation, err, args...)
	all := append(args, err)
	return apperror.MarkLogged(constants.ErrInternalWithMsg(fmt.Sprintf(msgFmt, all...)))
}

// InternalMsg 记录固定内部错误话术（无底层 err 变量时）。
func InternalMsg(ctx context.Context, component, operation, msg string, attrs ...any) error {
	attrs = append([]any{"operation", operation, "message", msg}, attrs...)
	biz(component).W(ctx).Error("Internal operation failed", attrs...)
	return apperror.MarkLogged(constants.ErrInternalWithMsg(msg))
}

// InternalFmt 按 fmt 拼内部错误话术并记日志（参数不必含 err）。
func InternalFmt(ctx context.Context, component, operation, msgFmt string, args ...any) error {
	msg := fmt.Sprintf(msgFmt, args...)
	return InternalMsg(ctx, component, operation, msg)
}

// Warn 记录 warn 级运维日志（可疑但非失败，如公开注册密钥错误）。
func Warn(ctx context.Context, component, operation, msg string, attrs ...any) {
	attrs = append([]any{"operation", operation}, attrs...)
	biz(component).W(ctx).Warn(msg, attrs...)
}

// Reject 记录预期的业务拒绝（warn，写入 info.log），返回 err（通常为 AppError）。
func Reject(ctx context.Context, component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	attrs = append([]any{"operation", operation, "error", err.Error()}, attrs...)
	biz(component).W(ctx).Warnw("Operation rejected", attrs...)
	return apperror.MarkLogged(err)
}

// Pass 记录非预期错误并原样返回（如 DB 驱动错误）；已是 AppError 时不再重复记 error。
func Pass(ctx context.Context, component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return apperror.MarkLogged(err)
	}
	biz(component).W(ctx).Op(operation, err, attrs...)
	return apperror.MarkLogged(err)
}
