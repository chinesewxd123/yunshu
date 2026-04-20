# 菜单需求：Deployment 管理（`/deployments`）

## 1. 定位

- **路由**：`/deployments`，`deployments-page`。  
- **目标**：Deployment **列表、详情、YAML 应用、扩缩容、重启、删除**及关联 Pod 列表。

## 2. API（`/api/v1/deployments`）

- `GET`、`GET /detail`、`GET /pods`  
- `POST /apply`、`/scale`、`/restart`  
- `DELETE`  

## 3. 注意事项

- `restart` 一般为重建 Pod 策略；生产需遵守发布窗口。  
- 所有请求需带 **cluster 上下文**（查询参数或 Header，以前端 `workloads` 封装为准）。
