# 菜单需求：日志源配置（`/project-log-sources`）

## 1. 定位

- **路由**：`/project-log-sources`，`project-log-sources-page`。  
- **目标**：为项目配置 **日志源**（文件路径、journal、编码等），供 Agent 采集与日志平台检索。

## 2. API

- `GET /api/v1/projects/:id/log-sources`  
- `POST /api/v1/projects/:id/log-sources`  
- `DELETE /api/v1/projects/:id/log-sources/:logSourceId`

## 3. 数据表

- `service_log_sources`（及与 `log_agents` 绑定关系）。

## 4. 注意事项

- 修改日志源可能影响已部署 Agent 的运行时拉取配置；注意滚动与兼容。
