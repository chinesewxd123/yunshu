# 菜单需求：RBAC — ClusterRoleBinding（`/rbac/clusterrolebindings`）

## 1. 定位

- **路由**：`/rbac/clusterrolebindings`，`rbac-clusterrolebindings-page`。  
- **目标**：**ClusterRoleBinding** 列表与维护。

## 2. API（`/api/v1/rbac`）

- `GET /clusterrolebindings`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 绑定集群管理员类高权限 ClusterRole 前必须双人复核。
