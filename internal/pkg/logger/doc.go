// Package logger 提供三通道结构化日志（log/slog），设计参考 onexstack/onex pkg/log，并适配 yunshu 四文件语义。
//
// # 文件分流（config.Log.file_path 目录）
//
//	info.log  — Info、Warn（业务流水、访问 4xx）
//	error.log — Error+（panic、5xx、SQL 错误）
//	sql.log   — 仅 GORM Trace
//
// # 与 onex 对齐的用法
//
//	logx.Init(app.Logger)                                    // 等同 onex log.Init
//	logx.W(ctx).WithLayer(logx.LayerService)                 // 等同 onex log.W(ctx)
//	logx.Biz("mysql.backup").W(ctx).Infow("started", "id", id)
//	logx.Biz("mysql.backup").Errorw(err, "upload failed")
//	logx.Sync()                                              // 进程退出前（可选）
//
// Service 层推荐：
//
//	svclog.ServiceCtx(ctx, "project").Infow("created", "project_id", id)
//	svcerr.Pass(ctx, "project", "Create", err)
//
// # 分层
//
//	layer: http | api | service | dao | grpc | worker
//	component: 模块名，如 mysql.backup、alert、k8s.event_forward
//
// HTTP 中间件自动写入 context：request_id（RequestLogger）、user_id/username（Auth）。
// GORM Trace 会从 ctx 附带 request_id 写入 sql.log。
//
// 禁止 fmt.Print / log.Println 打业务日志；审计走 login_logs / operation_logs 表。
package logger
