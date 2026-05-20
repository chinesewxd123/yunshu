// Package svclog 是业务侧统一日志入口（底层仍为 internal/pkg/logger）。
//
// 用法：
//   - 记流水：svclog.ServiceCtx(ctx, "user").Infow("Created user", "id", id)
//   - 后台任务：svclog.Worker("mysql.backup").Infow("Started scheduler", ...)
//   - 出错返回：svcerr.Pass(ctx, "user", "Create", err)（内部会记日志）
//
// 不要在新代码里使用 logx.Biz、结构体 bizLog 字段或 app.Logger.Biz。
package svclog

import (
	"context"

	logx "yunshu/internal/pkg/logger"
)

func withLayer(layer, name string) *logx.Component {
	return logx.Biz(name).WithLayer(layer)
}

// Service 返回 layer=service 的业务日志组件。
func Service(component string) *logx.Component {
	return withLayer(logx.LayerService, component)
}

// ServiceCtx 带 context 的 Service 日志（自动附加 request_id / user）。
func ServiceCtx(ctx context.Context, component string) *logx.Component {
	return Service(component).W(ctx)
}

// Worker 返回 layer=worker 的后台任务日志组件。
func Worker(component string) *logx.Component {
	return withLayer(logx.LayerWorker, component)
}

// WorkerCtx 带 context 的 Worker 日志。
func WorkerCtx(ctx context.Context, component string) *logx.Component {
	return Worker(component).W(ctx)
}

// HTTP 返回 layer=http 的日志（中间件、recovery）。
func HTTP(component string) *logx.Component {
	return withLayer(logx.LayerHTTP, component)
}

// HTTPCtx 带 context 的 HTTP 层日志。
func HTTPCtx(ctx context.Context, component string) *logx.Component {
	return HTTP(component).W(ctx)
}

// API 返回 layer=api 的日志（Handler / 写响应前的补充日志）。
func API(component string) *logx.Component {
	return withLayer(logx.LayerAPI, component)
}

// APICtx 带 context 的 API 层日志。
func APICtx(ctx context.Context, component string) *logx.Component {
	return API(component).W(ctx)
}

// GRPC 返回 layer=grpc 的日志。
func GRPC(component string) *logx.Component {
	return withLayer(logx.LayerGRPC, component)
}

// GRPPCtx 带 context 的 gRPC 层日志。
func GRPPCtx(ctx context.Context, component string) *logx.Component {
	return GRPC(component).W(ctx)
}
