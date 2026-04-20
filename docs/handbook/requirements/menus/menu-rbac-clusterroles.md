# 菜单需求：RBAC — ClusterRole（`/rbac/clusterroles`）

## 1. 定位

- **路由**：`/rbac/clusterroles`，`rbac-clusterroles-page`。  
- **目标**：集群级 **ClusterRole** 管理。

## 2. API（`/api/v1/rbac`）

- `GET /clusterroles`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- ClusterRole 影响范围为全集群，误配风险高于命名空间 Role。
