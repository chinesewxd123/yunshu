# 菜单需求：API 管理（`/permissions`）

## 1. 定位

- **路由**：`/permissions`，`PermissionsPage`。  
- **目标**：维护 **Permission** 元数据：`resource`（与 Gin `FullPath` 一致）+ `action`（HTTP 方法），供「授权管理」勾选。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/permissions`。 |
| 新建 API 定义 | `POST /api/v1/permissions`。 |
| 详情/编辑/删除 | `GET/PUT/DELETE /api/v1/permissions/:id`。 |

## 3. 数据表

- `permissions`；与 Casbin 策略行通过「授权」关联，非自动同步所有路由。

## 4. 注意事项

- **新增后端路由**后应运行 `seed` 或手工新增权限记录，否则非 super-admin 无法在「授权管理」中勾选。  
- `resource` 必须与运行时 `c.FullPath()` 一致（含 `:id` 形态）。  
- 全量路由清单见 `docs/apipost/permission-system.openapi.yaml`（genopenapi 生成）。
