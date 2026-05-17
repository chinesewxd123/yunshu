package server

import (
	"errors"
	"net/http"

	"yunshu/internal/pkg/apperror"
	logx "yunshu/internal/pkg/logger"
)

func logGRPCError(method string, err error) {
	if err == nil || apperror.AlreadyLogged(err) {
		return
	}
	b := logx.Biz("grpc")
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs := []any{"method", method, "error_code", appErr.ErrorCode, "http_status", appErr.StatusCode, "error", err.Error()}
		if appErr.StatusCode >= http.StatusInternalServerError {
			b.Error("rpc business error", attrs...)
			return
		}
		b.Debug("rpc client error", attrs...)
		return
	}
	b.Error("rpc failed", "method", method, "error", err.Error())
}
