# 菜单需求：Node 管理（`/nodes`）

## 1. 定位

- **路由**：`/nodes`，`nodes-page`。  
- **目标**：查看节点详情、**调度开关（cordon）**、**污点**管理。

## 2. API（`/api/v1/nodes`）

- `GET` 列表，`GET /detail`  
- `POST /schedulability`：调度/禁止调度  
- `PUT /taints`：替换污点  

## 3. 注意事项

- 污点误操作可能导致 Pod 无法调度；建议限制角色。  
- 部分路由未挂 `k8sScopeAuthorize`（以 `router.go` 为准），权限模型与 workload 页略有不同。
