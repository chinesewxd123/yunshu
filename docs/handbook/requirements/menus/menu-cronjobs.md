# 菜单需求：CronJob 管理（`/cronjobs`）

## 1. 定位

- **路由**：`/cronjobs`，`cronjobs-page`。  
- **目标**：CronJob 列表（含 v2 扩展字段）、详情、暂停/恢复、手动触发、YAML、删除及关联 Pod。

## 2. API（`/api/v1/cronjobs`）

- `GET`、`GET /v2`、`GET /detail`、`GET /pods`  
- `POST /apply`、`/suspend`、`/trigger`  
- `DELETE`  

## 3. 注意事项

- `trigger` 会立即创建 Job 实例；注意业务幂等与资源占用。
