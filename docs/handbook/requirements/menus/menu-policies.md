# 菜单需求：授权管理（`/policies`）

## 1. 定位

- **路由**：`/policies`，`PoliciesPage`。  
- **目标**：将 **API 权限**授予 **角色**（写入 Casbin），实现 RBAC。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 策略矩阵展示 | `GET /api/v1/policies`，合并角色、权限、Casbin 策略行。 |
| 授权 | `POST /api/v1/policies`，Body 含 `role_id` 与 `permission_id`（或项目约定字段）。 |
| 撤销 | `DELETE /api/v1/policies`，按 JSON body 指定策略（注意与历史路由一致）。 |

## 3. 注意事项

- 授权变更后服务内会 **同步用户 Casbin 关系**，用户需重新登录或等待同步策略方可感知（以实际 `SyncUserRoles` 调用点为准）。  
- **K8s 三元策略**在独立菜单维护，本页主要管 **REST API 级**权限。
