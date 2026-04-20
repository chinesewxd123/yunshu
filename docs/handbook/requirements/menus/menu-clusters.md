# 菜单需求：集群管理（`/clusters`）

## 1. 定位

- **路由**：`/clusters`，`ClusterPage`。  
- **目标**：维护平台纳管的 **Kubernetes 集群**连接信息（kubeconfig/地址等），为所有 K8s 资源页提供集群上下文。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/clusters`。 |
| 详情 | `GET /api/v1/clusters/:id`。 |
| 新建/更新 | `POST` / `PUT /api/v1/clusters/:id`。 |
| 删除 | `DELETE /api/v1/clusters/:id`。 |
| 启停 | `PUT /api/v1/clusters/:id/status`。 |
| 连接探测 | `GET /api/v1/clusters/:id/status`。 |
| 命名空间列表 | `GET /api/v1/clusters/:id/namespaces`。 |
| 控制面组件状态 | `GET /api/v1/clusters/:id/component-statuses`。 |

## 3. 数据表

- `k8s_clusters`。

## 4. 权限

- 路由使用 `k8sScopeAuthorize`：除 Casbin API 权限外，读操作可能受 **K8s 三元策略** 兜底（见权限手册）。

## 5. 注意事项

- kubeconfig **敏感**，存储与展示需脱敏；传输走 HTTPS。  
- 集群不可用会导致依赖该集群 ID 的页面批量失败。
