package handler

import logx "yunshu/internal/pkg/logger"

// SetLogger 注入进程默认业务日志（HTTP 经 response.Error → logHTTPError 写日志）。
func SetLogger(l *logx.Logger) {
	logx.SetDefault(l)
}
