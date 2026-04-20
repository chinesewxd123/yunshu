# 菜单需求：DaemonSet 管理（`/daemonsets`）

## 1. 定位

- **路由**：`/daemonsets`，`daemonsets-page`。  
- **目标**：DaemonSet 列表/详情/重启/YAML/删除及关联 Pod。

## 2. API（`/api/v1/daemonsets`）

- `GET`、`GET /detail`、`GET /pods`  
- `POST /apply`、`/restart`  
- `DELETE`  

## 3. 注意事项

- 典型用于日志、监控等节点级守护；重启会影响全集群节点上的 Pod。
