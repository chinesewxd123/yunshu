package server

import (
	"context"
	"errors"
	"net/http"

	"yunshu/internal/pkg/apperror"
	logx "yunshu/internal/pkg/logger"
)

func logGRPCError(ctx context.Context, method string, err error) {
	if err == nil || apperror.AlreadyLogged(err) {
		return
	}
	b := logx.Biz("grpc.server").WithLayer(logx.LayerGRPC).W(ctx)
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs := []any{"method", method, "error_code", appErr.ErrorCode, "reason", appErr.Reason, "http_status", appErr.StatusCode, "error", err}
		if appErr.StatusCode >= http.StatusInternalServerError {
			b.Errorw(err, "gRPC request failed", attrs...)
			return
		}
		b.Warnw("gRPC request rejected", append(attrs, "error", err.Error())...)
		return
	}
	b.Errorw(err, "gRPC request failed", "method", method)
}
