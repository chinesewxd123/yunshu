# 菜单需求：RBAC — Role（`/rbac/roles`）

## 1. 定位

- **路由**：`/rbac/roles`，`rbac-roles-page`。  
- **目标**：命名空间级 **Role** 与 **RoleBinding** 浏览/管理（与下面三个 RBAC 菜单共同使用 `rbac` API）。

## 2. API（`/api/v1/rbac`）

- `GET /roles`、`GET /rolebindings`、`GET /detail`  
- `POST /apply`、`DELETE`  

## 3. 注意事项

- 改 RBAC 会直接影响集群内授权；建议仅管理员操作并配合审计。
