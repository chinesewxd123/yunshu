# 菜单需求：封禁 IP 管理（`/banned-ips`）

## 1. 定位

- **路由**：`/banned-ips`，`BannedIPsPage`。  
- **目标**：查看因风控**被封禁的 IP** 并支持解封。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/security/banned-ips`。 |
| 解封 | `POST /api/v1/security/banned-ips/unban`。 |

## 3. 权限

- 路由挂载在 `security` 组，通常仅管理员使用；具体以 Casbin 为准。

## 4. 注意事项

- 与登录失败阈值、Redis 封禁逻辑配合；解封后该 IP 可立即重试登录。
