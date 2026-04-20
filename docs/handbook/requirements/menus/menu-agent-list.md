# 菜单需求：Agent 列表（`/agent-list`）

## 1. 定位

- **路由**：`/agent-list`，`agent-list-page`。  
- **目标**：项目下查看 **Log Agent** 心跳、版本、状态；支持刷新心跳、引导、轮换令牌等。

## 2. 主要 API

- `GET /api/v1/projects/:id/agents/list`  
- `POST .../agents/heartbeat-refresh`  
- `GET .../agents/status`  
- `POST .../agents/bootstrap`、`POST .../agents/rotate-token`  
- `GET .../agents/discovery`：发现数据。

## 3. 全局 Agent

- `POST /api/v1/agents/register`（鉴权）  
- `GET /api/v1/agents/runtime-config`（Agent 拉配置，公开性以路由中间件为准）  
- `POST /api/v1/agents/public-register`、`/health/report`、`/agents/discovery/report`：见 OpenAPI 安全标注。

## 4. 数据表

- `log_agents`、`agent_discovery`。

## 5. 注意事项

- Agent 二进制参数见部署手册与 `cmd/logagent`。  
- `register_secret` 需与配置一致。
