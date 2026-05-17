// Package logger 应用日志与 GORM SQL 日志适配。
//
// 业务层请使用 internal/service/svcerr（底层复用 Biz / LogErr），勿新建 slog 实例。
//
// 分层（运维排障日志，不是 CUD 审计）：
//
//	访问日志  — middleware.RequestLogger（每条 HTTP 一行）
//	业务错误  — svcerr.Pass / Internal（component + operation + error）
//	可疑行为  — svcerr.Warn / Reject（debug，如登录失败、非法注册）
//	API 边界  — response.logHTTPError（跳过 apperror.AlreadyLogged）
//	审计留痕  — login_logs、operation_logs 等数据库表，不写 info 级“创建成功”日志
//
// 文件：{file_path}/info.log、error.log、sql.log（由 config.LogConfig 控制）。
package logger
