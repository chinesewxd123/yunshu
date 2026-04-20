# 架构梳理与优化建议

本文基于当前仓库实现（Gin + GORM + MySQL + Redis + Casbin + gRPC + React/Ant Design）从**功能域、调用链、SQL/数据层**归纳现状与可优化点。

## 1. 功能域总览

| 域 | 后端入口 | 数据核心 | 说明 |
|----|----------|----------|------|
| 认证与账号 | `/api/v1/auth/*` | `users`、`roles`、`user_roles` | JWT、验证码、注册申请 |
| RBAC / 菜单 / 字典 | `/api/v1/roles`、`/permissions`、`/menus`、`/dict-entries` | `permissions`、`casbin_rule`（经适配器）、`menus`、`dict_entries` | API 权限 + Casbin；字典可覆盖邮件等配置 |
| 项目管理 | `/api/v1/projects/*` | `projects`、`project_members`、`servers`… | 租户边界；成员与告警通知联动 |
| 告警 | `/api/v1/alerts/*` | `alert_*` 系列表 | 通道、策略、数据源、静默、监控规则、处理人、值班班次 |
| K8s 运行时 | `/api/v1/clusters`、`/pods`… | `k8s_clusters` + 集群外资源 | 多集群连接；部分读接口可由 K8s 三元策略兜底 |
| K8s 三元策略 | `/api/v1/k8s-policies/*` | Casbin 中 `k8s:cluster:*` 对象 | 集群/命名空间/资源路径维度的授权 |
| 日志 Agent | gRPC + `/api/v1/projects/:id/agents/*` | `log_agents`、`service_log_sources` | Agent 注册、心跳、日志上报 |
| 运维审计 | `/api/v1/login-logs`、`/operation-logs` | `login_logs`、`operation_logs` | 登录与操作留痕 |

## 2. 典型调用链

### 2.1 HTTP 请求（业务 API）

1. `middleware.Auth`：解析 `Authorization: Bearer`，校验 Redis 会话（若启用）、加载用户与角色。
2. `middleware.Authorize`（Casbin）：`super-admin` 全放行；否则 `(user_id, FullPath, Method)`；部分 **GET** 在具备 K8s 三元策略时可放行（见 `middleware/casbin.go`）。
3. `middleware.OperationAudit`：写操作日志（视路由配置）。
4. Handler → Service → Repository → MySQL。

### 2.2 告警通知邮箱合并

`AlertRuleAssigneeService.ResolveNotifyEmails`：处理人（用户/部门 JSON）→ 用户邮箱；规则所属项目由数据源派生，再合并 **项目成员** 中启用且有邮箱的用户（去重）。与 `project_members` 表对齐。

### 2.3 gRPC（日志/Agent）

平台侧 `grpc/listen_addr`（默认 `18080`）与 HTTP `app.port`（默认 `8080`）分离；Agent 与日志相关服务走 gRPC，管理面走 HTTP。

## 3. SQL 与数据层优化建议

1. **索引与查询**
   - 列表接口普遍带 `keyword`、外键 `project_id`、`monitor_rule_id` 等，建议在业务增长后按慢查询日志补复合索引（如 `project_members(project_id, deleted_at)` 已有唯一场景由 GORM 维护）。
   - 大表（`alert_events`、`operation_logs`、`login_logs`）建议定期归档或按时间分区（需 DBA 评估）。

2. **N+1**
   - 列表若逐条 `GetByID`，应改为 `WHERE id IN (...)` 批量查询；当前部分模块已用联表（如项目成员列表）。

3. **事务边界**
   - 「创建项目 + owner 成员」已在成员写入失败时软删回滚项目，避免脏数据；其它多表写操作可按业务需要引入显式事务（`db.Transaction`）。

4. **字典与配置**
   - 邮件等配置优先字典、YAML 兜底，避免双源不一致；启动迁移含字典去重 SQL（`migrate_schema.go`）。

5. **Redis**
   - 会话与限流依赖 Redis，生产需高可用；JWT 黑名单/会话键需与 TTL 策略一致。

## 4. 代码结构优化建议

1. **Swagger/OpenAPI 与实现同步**：路由变更后重新生成 Swagger，或主维护 `docs/apipost/*.yaml`。
2. **前端**：大页（如告警监控平台）可继续按 Tab 拆子路由或 lazy chunk，控制首包体积。
3. **权限种子**：新增 API 需在 `cmd/seed.go` 的 `defaultPermissions()` 增加条目，否则非 `super-admin` 需在「授权管理」手工勾选。

## 5. 安全与合规

- 生产必须替换 `jwt_secret`、`encryption_key`、`agent.register_secret` 等。
- 服务器凭证、云账号密钥经 AES-GCM 存储，密钥长度需符合配置说明。

---

*文档版本与仓库同步；若与代码冲突以源码为准。*
