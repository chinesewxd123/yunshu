# 日志平台接口说明（Log Platform API）

本文档覆盖当前代码中**已实现**的日志平台相关接口与协议，面向前端、Agent、后端联调与运维部署。

- 代码基线：`internal/router/router.go` 当前注册路由
- 适用范围：项目日志源、Agent 注册与配置、日志上报、日志流消费、发现能力

---

## 1. 认证与访问约定

### 1.1 平台管理接口（`/projects/*`）

- 需要业务登录态（JWT）
- 经过鉴权中间件：`authMiddleware + authorize + opAudit`

### 1.2 Agent 侧接口（HTTP 兼容 + gRPC 主链路）

- HTTP 兼容入口：
  - `POST /api/v1/agents/public-register`
  - `GET /api/v1/agents/runtime-config`
  - `POST /api/v1/agents/discovery/report`
- Agent 主链路：
  - gRPC `AgentRuntimeService/PublicRegister`
  - gRPC `AgentRuntimeService/GetRuntimeConfig`
  - gRPC `AgentRuntimeService/ReportDiscovery`
  - gRPC `AgentRuntimeService/IngestLogs`（双向流）

> 说明：Agent token 为长效密钥，服务端保存其 hash（SHA-256）。

---

## 2. 数据模型（关键）

### 2.1 LogSource（日志源）

- `id`：日志源 ID（即 `log_source_id`）
- `log_type`：`file` 或 `journal`
- `path`：路径、目录或 glob（例如 `/var/log/pods/.../*.log`）

### 2.2 Stream Key

服务端 Broker 使用如下 key 聚合流：

```text
<project_id>:<server_id>:<log_source_id>
```

### 2.3 Agent 上报事件（推荐）

当前推荐上报格式（支持文件路径维度）：

```json
{
  "project_id": 1,
  "server_id": 1,
  "log_source_id": 9,
  "seq": 101,
  "entries": [
    { "line": "2026-04-15T15:00:01Z info ...", "file_path": "/var/log/pods/.../0.log" },
    { "line": "2026-04-15T15:00:02Z warn ...", "file_path": "/var/log/pods/.../1.log" }
  ]
}
```

兼容字段（历史）：

- `line` + `file_path`
- `lines[]` + `file_paths[]`

---

## 3. 项目日志源接口（平台侧）

Base: `/api/v1/projects/:id`

### 3.1 查询日志源

- **GET** `/:id/log-sources`
- Query：`service_id`、`page`、`page_size`
- 用途：日志平台页“日志源”下拉数据

### 3.2 新增/更新日志源（Upsert）

- **POST** `/:id/log-sources`
- Body（示例）：

```json
{
  "id": 9,
  "service_id": 3,
  "log_type": "file",
  "path": "/var/log/pods/.../*.log",
  "status": 1
}
```

> `id` 为空表示创建；有值表示更新。

### 3.3 删除日志源

- **DELETE** `/:id/log-sources/:logSourceId`

---

## 4. Agent 管理接口（平台侧）

Base: `/api/v1/projects/:id`

### 4.1 Agent 状态

- **GET** `/:id/agents/status`
- Query：
  - `server_id`（required）
  - `log_source_id`（optional）

响应关键字段：

- `online`：是否在线（最近心跳 90s）
- `recent_publishing`：最近是否有日志上报（30s窗口）
- `last_seen_at`

### 4.2 生成部署命令（Bootstrap）

- **POST** `/:id/agents/bootstrap`
- Body：

```json
{
  "server_id": 1,
  "platform_url": "http://10.10.10.1:8080",
  "agent_name": "log-agent",
  "agent_version": "v0.1.0"
}
```

返回：

- `token`
- `run_command`
- `systemd_service`

### 4.3 轮换 Token

- **POST** `/:id/agents/rotate-token`
- 入参与 bootstrap 相同
- 返回新 token + 新部署命令

### 4.4 查询 Agent Discovery（文件下拉来源）

- **GET** `/:id/agents/discovery`
- Query：
  - `server_id`（required）
  - `kind`（optional：`file|dir|unit`）
  - `limit`（optional）

返回列表项：

- `kind`
- `value`
- `extra`
- `last_seen_at`

---

## 5. Agent 运行时接口（Agent 调用）

### 5.1 公共注册（可选）

- **POST** `/api/v1/agents/public-register`
- Body：

```json
{
  "server_id": 1,
  "name": "log-agent",
  "version": "v0.1.0",
  "register_secret": "your-secret"
}
```

返回：

```json
{
  "project_id": 1,
  "agent_id": 12,
  "token": "xxxx"
}
```

### 5.2 拉取运行配置

- **GET** `/api/v1/agents/runtime-config?token=<token>`

返回：

```json
{
  "project_id": 1,
  "server_id": 1,
  "sources": [
    { "log_source_id": 1, "log_type": "file", "path": "/var/log/messages" },
    { "log_source_id": 9, "log_type": "file", "path": "/var/log/pods/.../*.log" }
  ]
}
```

### 5.3 上报发现结果（Discovery）

- **POST** `/api/v1/agents/discovery/report`
- Body：

```json
{
  "token": "xxxx",
  "items": [
    { "kind": "file", "value": "/var/log/pods/.../0.log" },
    { "kind": "dir", "value": "/var/log/pods/..." },
    { "kind": "unit", "value": "kubelet.service" }
  ]
}
```

返回：

```json
{ "accepted": 123 }
```

### 5.4 gRPC 日志上报（主链路）

- **RPC** `AgentRuntimeService/IngestLogs`（bidirectional stream）
- Agent 发消息：见“2.3 Agent 上报事件”（映射为 `IngestLogsRequest.entries`）
- Server ACK：

```json
{ "seq": 101, "ts_unix_ms": 1713170000000 }
```

---

## 6. 日志流消费接口（前端 SSE）

### 6.1 实时流

- **GET** `/api/v1/projects/:id/logs/stream`
- SSE 响应：`event: log` + `event: ping`

Query 参数：

- `server_id`（required）
- `log_source_id`（required）
- `tail_lines`（optional，<=0 时默认 200；用于历史回放行数）
- `include`（optional，regex）
- `exclude`（optional，regex）
- `highlight`（optional，关键字高亮）
- `file_path`（optional，按具体文件过滤）

SSE `log` 数据示例：

```json
{
  "line": "2026-04-15 15:55:14 scheduler started",
  "file_path": "/var/log/pods/.../kube-scheduler/0.log"
}
```

> 行长度超过 4096 时会截断并追加 `...<truncated>`。

### 6.2 历史回放行为

- 订阅建立后先从内存历史缓冲回放最近 `tail_lines` 行
- 然后进入实时推送
- 每个 stream key 历史上限：5000 行（服务端内存）

### 6.3 导出接口（当前状态）

- **GET** `/api/v1/projects/:id/logs/export`
- 当前实现返回错误：仅支持实时 agent stream，不支持旧 SSH 导出链路

---

## 7. 错误语义（常见）

### 7.1 Agent 侧

- `token is required (or provide register-secret...)`
- `runtime-config failed: status=...`
- `no runtime sources from server and no fallback log-source-id/path provided`
- `invalid agent token`

### 7.2 流接口

- `invalid include regex`
- `invalid exclude regex`
- `server not in project`

### 7.3 前端常见表现

- SSE 黑屏但无报错：
  - 可能是当前无新写入；可依赖 tail 回放验证
  - 或 filter 条件把内容过滤掉

---

## 8. 联调最小闭环

1. 平台配置 `project/server/service/log-source`
2. 获取 token（bootstrap/rotate）
3. 启动 agent 并确认：
   - runtime-config 成功
   - ingest stream connected
4. 打开日志平台页面请求 SSE：
   - 参数带 `server_id + log_source_id`
5. 验证：
   - 有历史回放（tail_lines）
   - 有实时新增
   - `file_path` 下拉筛选生效

---

## 9. Agent 启动命令（3种常用模式）

以下命令基于当前 `log-agent` 参数实现，可直接用于 Linux 主机。

### 9.1 模式 A：Token 标准模式（推荐）

适用于：平台已生成 token、日志源由平台下发。

```bash
./log-agent \
  --grpc-server "10.10.10.1:18080" \
  --project-id 1 \
  --server-id 1 \
  --token "xxxxxxxxxxxxxxxx"
```

### 9.2 模式 B：Token + 调试模式

适用于：联调/排障，需要查看采集与发送细节。

```bash
./log-agent \
  --grpc-server "10.10.10.1:18080" \
  --project-id 1 \
  --server-id 1 \
  --token "xxxxxxxxxxxxxxxx" \
  --debug
```

### 9.3 模式 C：Fallback 单日志源模式

适用于：runtime-config 拉取失败时，临时指定单源采集。

```bash
./log-agent \
  --grpc-server "10.10.10.1:18080" \
  --project-id 1 \
  --server-id 1 \
  --token "xxxxxxxxxxxxxxxx" \
  --log-source-id 9 \
  --source-type file \
  --path "/var/log/pods/.../*.log" \
  --tail-lines 300
```

> 说明：`--log-source-id + --path` 仅在 fallback 场景需要，生产建议优先使用平台下发配置。

---

## 10. 版本兼容建议

- Agent 上报优先使用 `entries[]`（推荐）
- 服务端 ingest 主链路为 gRPC stream（HTTP 仅保留兼容入口）
- 升级时建议平台与 agent 同步发布

