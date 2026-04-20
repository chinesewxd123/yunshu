# 需求说明：项目管理

## 1. 目标

以 **Project** 为租户边界，管理服务器、服务元数据、日志源、云同步及 **项目成员**；与告警监控中的「项目归属（由数据源派生）」与通知收件人联动。

## 2. 功能结构

```
项目管理
├── 项目列表：CRUD、编码唯一
├── 项目成员：用户 + 项目内角色 owner/admin/member/readonly（独立于全局 RBAC）
├── 服务器管理：分组、导入导出、连通性、终端 WebSocket、云账号同步
├── 服务配置：项目下业务服务注册
├── 日志源配置：文件/journal 等与 Agent 绑定
├── 日志平台：日志检索、流式导出（HTTP + gRPC）
└── Agent 列表：心跳、令牌轮换、发现上报
```

## 3. 子功能与注意事项

| 子功能 | 说明 |
|--------|------|
| 创建项目 | 创建人自动写入 `project_members` 为 **owner**；失败会回滚项目记录 |
| 成员 | 同一 `(project_id, user_id)` 唯一；成员邮箱可并入“该项目下数据源对应的监控规则”通知 |
| 服务器密钥 | 使用配置项 `security.encryption_key` 加密存储 |
| 终端 | WebSocket 路径需单独鉴权，与 HTTP 策略一致 |

## 4. 相关表

`projects`、`project_members`、`server_groups`、`servers`、`server_credentials`、`cloud_accounts`、`services`、`service_log_sources`、`log_agents`、`agent_discovery`。

## 5. API 前缀

`/api/v1/projects/...`（详见接口文档与 `cmd/seed.go` 权限种子）。
