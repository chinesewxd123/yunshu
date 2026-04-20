# 菜单需求：账号管理（`/users`）

## 1. 定位

- **路由**：`/users`，`UsersPage`，顶栏「系统管理」或种子菜单「账号管理」。  
- **目标**：维护平台登录账号生命周期，与部门、全局角色绑定；支持 Excel 导入导出。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 分页列表 | `GET /api/v1/users`，支持关键词（用户名/昵称等）。 |
| 新建用户 | `POST /api/v1/users`，密码策略与校验由后端 validator 决定。 |
| 详情 | `GET /api/v1/users/:id`。 |
| 编辑/禁用 | `PUT /api/v1/users/:id`。 |
| 删除 | `DELETE /api/v1/users/:id`（视业务是否软删）。 |
| 分配角色 | `PUT /api/v1/users/:id/roles`，同步 Casbin。 |
| 导出 | `GET /api/v1/users/export`。 |
| 导入 | `POST /api/v1/users/import`，模板 `GET /api/v1/users/import-template`。 |

## 3. 数据表

- `users`、`user_roles`、`departments`（外键展示）。

## 4. 权限

- 各接口需在 Casbin 中授权；种子见 `defaultPermissions` 中 `/api/v1/users` 系列。

## 5. 注意事项

- **告警/项目成员**等模块通过用户邮箱发通知，用户需维护正确邮箱与启用状态。  
- 修改角色会触发 `SyncUserRoles`，权限变更即时影响 API 访问。  
- 导入大批量时注意超时与事务大小，必要时分批。
