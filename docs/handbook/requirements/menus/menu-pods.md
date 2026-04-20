# 菜单需求：Pod 管理（`/pods`）

## 1. 定位

- **路由**：`/pods`，`PodPage`。  
- **目标**：在选定集群/命名空间下 **列出 Pod、查看详情、事件、日志、文件、Exec、终端 WebSocket、重启、YAML 创建/编辑**。

## 2. 主要 API（`/api/v1/pods`）

| 能力 | 路径示例 |
|------|----------|
| 列表/详情/事件 | `GET ""`、`/detail`、`/events` |
| 日志与下载 | `GET /logs`、`/logs/download`、`/logs/stream` |
| 文件 | `GET /files`、`/file`、`/file/download`，`POST /file/upload`、`/file/delete` |
| Exec | `POST /exec`，`GET /exec/ws`（WS 鉴权见中间件） |
| 生命周期 | `POST /restart`、`/create/yaml`、`/create/simple`、`/update/simple`，`DELETE` |

## 3. 权限

- **K8s 三元策略** + API 权限；Exec/WS 为高危操作需严格控制角色。

## 4. 注意事项

- 日志流与 Exec 长连接注意网关 **超时与 WebSocket** 配置。  
- 删除 Pod 为破坏性操作，需二次确认。
