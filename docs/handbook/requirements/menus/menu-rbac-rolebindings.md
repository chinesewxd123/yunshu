# 菜单需求：RBAC — RoleBinding（`/rbac/rolebindings`）

## 1. 定位

- **路由**：`/rbac/rolebindings`，`rbac-rolebindings-page`。  
- **目标**：**RoleBinding** 列表与详情、YAML 应用、删除。

## 2. API

- 同 `/api/v1/rbac` 下 `rolebindings` 与 `apply`/`delete`。

## 3. 注意事项

- 绑定变更立即影响主体（User/Group/ServiceAccount）权限。
