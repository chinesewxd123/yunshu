# 页面需求：登录（`/login`）

## 1. 定位

- **路由**：`/login`，未登录可访问；已登录会重定向到 `/`。  
- **组件**：`login-page`。  
- **目标**：用户名密码、邮箱验证码等多种方式换取 JWT，并写入前端存储/Context。

## 2. 功能点

| 功能 | 说明 |
|------|------|
| 用户名密码登录 | `POST /api/v1/auth/login`，成功返回 `access_token` 与用户基本信息。 |
| 邮箱验证码登录 | `POST /api/v1/auth/email-login`（以实际前端调用为准）。 |
| 发码 | `POST /api/v1/auth/verification-code`、`login-code`、`password-login-code` 等，受冷却时间限制。 |
| 注册申请 | 入口可链到注册页或弹窗，对应 `POST /api/v1/auth/register`。 |

## 3. 安全与合规

- **HTTPS**：生产环境必须，防止密码与 Token 明文传输。  
- **JWT**：`Authorization: Bearer`；服务端可配合 Redis 会话键使用户登出立即失效。  
- **审计**：登录成功/失败写入 `login_logs`（由服务端中间件或认证服务负责）。

## 4. 注意事项

- 默认种子账号见 `cmd/seed.go` 输出，**上线后必须改密与限制种子执行环境**。  
- 多次失败可能触发 IP 封禁（见封禁 IP 模块）。
