# 菜单需求：服务器管理（`/project-servers`）

## 1. 定位

- **路由**：`/project-servers`，`project-servers-page`。  
- **目标**：在**选定项目**下维护主机资产：分组、SSH/凭据、导入导出、连通性测试、云账号同步等。

## 2. 主要 API（前缀 `/api/v1/projects/:id`）

| 能力 | 方法 |
|------|------|
| 服务器列表/增改 | `GET/POST .../servers`，详情 `GET .../servers/:serverId`，删 `DELETE` |
| 分组树 | `GET .../server-groups/tree`，增改删 `POST/PUT/DELETE .../server-groups` |
| 云账号 | `GET/POST .../cloud-accounts`，更新/同步 `PUT .../sync` |
| 导入导出 | `POST .../servers/import`，`GET .../import-template`，`GET .../export` |
| 测试 | `POST .../servers/test`，`.../test/batch`，`POST .../servers/sync` |

## 3. 数据表

- `servers`、`server_groups`、`server_credentials`、`cloud_accounts` 等。

## 4. 注意事项

- 凭据使用 `security.encryption_key` 加密存储。  
- **服务器终端**见 `server-console` 与 WebSocket 文档。
