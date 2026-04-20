# 菜单需求：服务器操作台（`/server-console`）

## 1. 定位

- **路由**：`/server-console`，`server-console-page`。  
- **目标**：在已录入的 **项目服务器**上执行命令或进入 **Web 终端**（与 `projects` 下服务器资源对应）。

## 2. 主要 API

- 命令：`POST /api/v1/projects/:id/servers/:serverId/exec`  
- 终端：`GET /api/v1/projects/:id/servers/:serverId/terminal/ws`（WebSocket，`wsAuthMiddleware`）

## 3. 注意事项

- 需先选项目与服务器；凭据解密在后端完成。  
- 生产环境应限制可执行命令或审计全量会话。
