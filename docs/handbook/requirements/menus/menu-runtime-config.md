# 菜单需求：运行时配置（`/runtime-config`）

## 1. 定位

- **路由**：`/runtime-config`，`runtime-config-page`（动态菜单或运维入口）。  
- **目标**：查看/调整与 **Agent、gRPC、平台运行时** 相关的可读配置（具体字段以前端页与后端 `runtime-config` 接口为准）。

## 2. API

- 常与 `GET /api/v1/agents/runtime-config` 及项目侧 Agent 引导接口配合；详见前端 `runtime-config-page` 与 OpenAPI 中 `agents` 相关路径。

## 3. 注意事项

- 与 `configs/config.yaml` 静态配置区分：**运行时**项可能来自 DB 或字典。  
- 变更后需观察 Agent 重连与日志。
