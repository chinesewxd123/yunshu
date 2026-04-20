# 菜单需求：ConfigMap 管理（`/configmaps`）

## 1. 定位

- **路由**：`/configmaps`，`configmaps-page`。  
- **目标**：ConfigMap 列表、详情、YAML 应用、删除。

## 2. API（`/api/v1/configmaps`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 修改 ConfigMap 可能影响已挂载 Pod，需滚动更新或等待应用热加载。
