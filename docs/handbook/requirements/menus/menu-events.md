# 菜单需求：Event 事件（`/events`）

## 1. 定位

- **路由**：`/events`，`events-page`。  
- **目标**：集群/命名空间维度 **Kubernetes Events** 列表，用于排查调度、镜像拉取、探针失败等。

## 2. API（`/api/v1/events`）

- `GET` 列表（查询参数含集群、namespace 等，以前端为准）。

## 3. 注意事项

- Events 有保留时间，默认可能较短；与 `kubectl get events` 一致为排障辅助。
