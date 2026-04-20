# 菜单需求：角色管理（`/roles`）

## 1. 定位

- **路由**：`/roles`，`RolesPage`。  
- **目标**：维护**角色**定义（名称、编码如 `super-admin`），为授权管理与用户绑定提供主体。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/roles`。 |
| 新建 | `POST /api/v1/roles`。 |
| 详情 | `GET /api/v1/roles/:id`。 |
| 更新 | `PUT /api/v1/roles/:id`。 |
| 删除 | `DELETE /api/v1/roles/:id`（内置角色需保护）。 |

## 3. 数据表

- `roles`；关联 `user_roles`、Casbin 策略中的角色码。

## 4. 注意事项

- **`super-admin`** 在中间件层全放行，删除或改码可能导致无法管理。  
- 角色与 **API 权限**通过「授权管理」绑定，非改角色即自动拥有全部接口。
