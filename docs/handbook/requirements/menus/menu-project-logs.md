# 菜单需求：日志平台（`/project-logs`）

## 1. 定位

- **路由**：`/project-logs`，`project-logs-page`。  
- **目标**：在项目维度 **检索、流式浏览、导出** 日志（HTTP 侧与 gRPC 上报配合）。

## 2. 主要 API（前缀 `/api/v1/projects/:id`）

- `GET .../logs/stream`：流式。  
- `GET .../logs/export`：导出。  
- `GET .../log-files`、`GET .../log-units`：文件级/单元级浏览。

## 3. 注意事项

- 大结果集注意超时与浏览器内存；生产建议限制时间范围。  
- 与 **gRPC 日志上报**协议见 `docs/log-platform-api.md`。
