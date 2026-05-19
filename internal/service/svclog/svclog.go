// Package svclog 为 Service / Worker 层提供统一的业务日志组件（layer + component）。
package svclog

import (
	"context"

	logx "yunshu/internal/pkg/logger"
)

// Service 返回 layer=service 的业务日志组件。
func Service(component string) *logx.Component {
	return logx.Biz(component).WithLayer(logx.LayerService)
}

// ServiceCtx 带 context 的 Service 日志（自动附加 request_id / user）。
func ServiceCtx(ctx context.Context, component string) *logx.Component {
	return Service(component).W(ctx)
}

// Worker 返回 layer=worker 的后台任务日志组件。
func Worker(component string) *logx.Component {
	return logx.Biz(component).WithLayer(logx.LayerWorker)
}

// WorkerCtx 带 context 的 Worker 日志。
func WorkerCtx(ctx context.Context, component string) *logx.Component {
	return Worker(component).W(ctx)
}
