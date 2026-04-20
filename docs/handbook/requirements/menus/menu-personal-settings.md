# 菜单需求：个人设置（`/personal-settings`）

## 1. 定位

- **路由**：`/personal-settings`，`PersonalSettingsPage`。  
- **目标**：当前登录用户修改 **昵称、邮箱、密码** 等（不经过用户管理权限）。

## 2. 主要 API

- `GET /api/v1/auth/me`  
- `PUT /api/v1/auth/me`  
- `PUT /api/v1/auth/password`

## 3. 注意事项

- 修改邮箱会影响**告警通知**等依赖邮箱的功能。  
- 与管理员在「账号管理」中改他人资料区分权限边界。
