# 菜单需求：组件状态（`/component-status`）

## 1. 定位

- **路由**：`/component-status`，`component-status-page`。  
- **目标**：查看集群 **控制平面组件健康**（etcd、scheduler 等，取决于后端实现与集群版本）。

## 2. API

- `GET /api/v1/clusters/:id/component-statuses`（需先选集群 ID，与页面参数一致）。

## 3. 注意事项

- 仅作运维参考；异常需结合集群真实日志与云厂商面板。
