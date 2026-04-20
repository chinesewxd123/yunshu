# 需求说明：认证与身份

## 1. 目标

为运维/管理员提供安全登录能力，支持用户名密码、邮箱验证码等模式；新用户通过**注册申请**由管理员审核。

## 2. 功能列表

| 功能 | 子功能 | 注意事项 |
|------|--------|----------|
| 登录 | 用户名密码、邮箱验证码、密码+验证码 | JWT 存 Redis 时登出即失效；`Authorization: Bearer` |
| 注册 | 提交申请 | 非直接开户，走 `registrations` 审核流 |
| 验证码 | 发邮件验证码 | 频率受 `auth.email_code_cooldown_seconds` 限制 |
| 个人设置 | 修改资料、密码 | 与 `users` 表一致 |

## 3. 数据与接口

- 表：`users`、`login_logs`、注册相关 `registration_requests`。
- 路由：`/api/v1/auth/*`（公开）、用户相关 `/api/v1/users/*`（鉴权）。

## 4. 非功能

- 生产必须修改 `jwt_secret`。
- 登录失败与封禁策略见安全模块与 `banned-ips`。
