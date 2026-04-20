# HTTP 接口约定（详细版摘要）

**Base URL**：`/api/v1`（以 `internal/router/router.go` 为准）  
**鉴权**：`Authorization: Bearer <access_token>`（除登录、健康检查、部分 Webhook 等公开接口）

## 1. 统一响应体

实现：`internal/pkg/response/response.go`

### 1.1 成功（HTTP 200）

```json
{
  "code": 200,
  "message": "success",
  "data": { }
}
```

### 1.2 创建成功（部分接口 HTTP 201）

```json
{
  "code": 201,
  "message": "success",
  "data": { }
}
```

### 1.3 业务/客户端错误

```json
{
  "code": 400,
  "error_code": "BAD_REQUEST",
  "message": "人类可读说明"
}
```

### 1.4 未授权 / 禁止

- `401`：未登录或 Token 无效  
- `403`：Casbin 拒绝或业务 Forbidden  

## 2. 分页列表（常见）

查询参数（示例，以具体 Handler 为准）：

| 参数 | 含义 |
|------|------|
| `page` | 页码，从 1 开始 |
| `page_size` / `pageSize` | 每页条数 |
| `keyword` | 关键词 |

响应 `data` 常见结构：

```json
{
  "list": [ ],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

实现类型参考：`internal/pkg/pagination.Result`。

## 3. 示例：登录

**请求** `POST /api/v1/auth/login`

```json
{
  "username": "admin",
  "password": "***"
}
```

**响应 `data`**（简化）

```json
{
  "access_token": "eyJ...",
  "expires_in": 7200,
  "user": {
    "id": 1,
    "username": "admin",
    "nickname": "..."
  }
}
```

## 4. 示例：创建项目

**请求** `POST /api/v1/projects`  
**Header**：`Authorization: Bearer ...`

```json
{
  "name": "生产项目",
  "code": "prod",
  "description": "可选",
  "status": 1
}
```

**响应**：`data` 为项目 DTO；服务端会将**当前登录用户**写入 `project_members` 为 `owner`。

## 5. 示例：项目成员列表

**请求** `GET /api/v1/projects/:id/members`

**响应**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "user_id": 2,
        "username": "zhang",
        "nickname": "张",
        "email": "a@b.com",
        "role": "owner",
        "created_at": "2026-01-01T00:00:00Z"
      }
    ]
  }
}
```

## 6. 完整路径清单

- **权限种子**：`cmd/seed.go` → `defaultPermissions()`（资源路径与 Method 与 Casbin 一致）。  
- **机器可读**：`docs/apipost/permission-system.openapi.yaml`、`docs/swagger/swagger.yaml`。  
- **在线文档**：若开启 `swagger.enabled`，访问配置项 `swagger.path`（默认 `/swagger`）。

## 7. WebSocket

- 例如 `GET /api/v1/projects/:id/servers/:serverId/terminal/ws`：使用 **查询参数或子协议** 传递 Token（见 `middleware/ws_auth` 实现）。
