# 菜单需求：StatefulSet 管理（`/statefulsets`）

## 1. 定位

- **路由**：`/statefulsets`，`statefulsets-page`。  
- **目标**：StatefulSet 列表/详情/扩缩容/重启/YAML/删除及关联 Pod。

## 2. API（`/api/v1/statefulsets`）

- `GET`、`GET /detail`、`GET /pods`  
- `POST /apply`、`/scale`、`/restart`  
- `DELETE`  

## 3. 注意事项

- 有状态应用扩缩容与 PVC 顺序有关，需谨慎。
