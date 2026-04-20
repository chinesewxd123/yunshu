# 菜单需求：Job 管理（`/jobs`）

## 1. 定位

- **路由**：`/jobs`，`jobs-page`。  
- **目标**：Job 列表、详情、**重新执行**、YAML、删除及关联 Pod。

## 2. API（`/api/v1/jobs`）

- `GET`、`GET /detail`、`GET /pods`  
- `POST /rerun`、`/apply`  
- `DELETE`  

## 3. 注意事项

- `rerun` 可能产生重复业务副作用，需业务侧可重入。
