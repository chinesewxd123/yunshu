---
name: yunshu-backend-dev
description: Guides Go backend development, API changes, and bugfixes in the yunshu repository following existing Gin/GORM/service patterns. Use when editing internal handlers or services, fixing HTTP/API bugs, adding routes, or debugging alert/K8s/project features in this codebase.
disable-model-invocation: true
---

# Yunshu 后端开发与 Bug 修复

## 仓库结构（改动前先定点）

| 区域 | 路径 | 说明 |
|------|------|------|
| 入口 | `cmd/server.go`、`cmd/root.go` | 服务启动、子命令 |
| 组装 | `internal/bootstrap/` | DB、Redis、配置、路由注册 |
| HTTP | `internal/router/router.go` | 路由分组、中间件挂载顺序 |
| 控制器 | `internal/handler/` | 解析参数、调用 service、统一响应 |
| 业务 | `internal/service/` | 事务边界、领域逻辑 |
| 模型 | `internal/model/` | GORM 模型、`TableName` |
| 仓储（若有） | `internal/repository/` | 数据访问薄封装 |
| 中间件 | `internal/middleware/` | 鉴权、审计、K8s 作用域 |
| 公共包 | `internal/pkg/` | `apperror`、`response`、工具 |
| 配置 | `internal/config/` | viper 映射字段 |

新接口：**先在 `router.go` 找准分组与前缀**（如 `/api/v1/alerts`），再在对应 `handler` + `service` 扩展；避免在未理清分组时复制粘贴路由。

## 编码惯例（与现有代码对齐）

1. **错误**：业务可预期错误用 `internal/pkg/apperror`（如 `apperror.BadRequest`、`NotFound`、`Forbidden`、`Internal`）；不要把字符串裸返回给 handler。
2. **HTTP 响应**：成功用 `response.Success`；失败用 `response.Error(c, err)`，让统一包装处理 `apperror`。
3. **Handler**：优先复用 `internal/handler/handler_exec.go` 里的 `handleJSON`、`handleJSONOK`、`handleQuery` 等，减少重复的 `ShouldBindJSON` / 错误分支。
4. **Context**：service 方法第一个参数传 `context.Context`，数据库用 `s.db.WithContext(ctx)`。
5. **GORM**：软删字段 `deleted_at`；列表查询注意 `Where("deleted_at IS NULL)` 或与模型一致；批量时注意 N+1。
6. **注释与文案**：用户可见错误信息可用中文；代码注释保持与文件现有风格一致，不为显而易见的逻辑写长注释。
7. **改动范围**：只改完成任务所需的文件与行；禁止顺带大范围格式化或无关重构。

## 新 API / 行为调整清单

- [ ] `router.go`：路径、HTTP 方法、`authMiddleware` / `authorize` / `opAudit` 是否与同级接口一致  
- [ ] `casbin_rule` / 权限种子：若前台菜单或 RBAC 依赖新路径，检查是否需在 `cmd/seed.go` 或迁移中注册权限（参考现有 `alerts/*` 条目）  
- [ ] Swagger：若项目维护 OpenAPI，同步 `tools/genopenapi` 或注释约定  
- [ ] 配置：新开关放入 `internal/config`，默认值与安全默认值明确  

## Bug 修复流程

1. **复现**：最小请求（curl / 前端操作）、期望与实际；区分 4xx（参数/权限）与 5xx。  
2. **链路**：Router → Handler → Service → DB/Redis/外部 API；对告警类问题可同时查 `alert_events`、`receiver_group`、`subscription_nodes`。  
3. **数据**：确认标签、`project_id`、软删、缓存（如接收组/订阅树缓存是否需失效或重启）。  
4. **验证**：`go build ./...`；若有 SQL 迁移，`migrate` 相关命令与 README。  
5. **回归**：相邻调用路径是否被破坏（例如同一 service 多入口）。

## 告警子域提示

订阅树、Webhook、`channel_ids_json` 等排障与 SQL 修正见同目录技能 **`yunshu-alert-receiver-bindings`**；本技能侧重通用后端改动纪律。

## 避免事项

- 在 handler 里堆核心业务逻辑。  
- 忽略路由中间件导致「本地可调、线上 403」。  
- 修改共享 model 字段却不迁移或不兼容旧前端。  
- 假设 Redis 一定可用（监控规则评估等路径对 nil Redis 行为敏感，改动前先读现有分支）。
