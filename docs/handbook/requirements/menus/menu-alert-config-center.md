# 菜单需求：告警配置中心（`/alert-config-center`）

## 1. 定位

- **路由**：`/alert-config-center`，`alert-config-center-page`（可由动态菜单或书签进入）。  
- **目标**：**订阅树/接收组**的配置管理与 **历史告警事件**检索；顶部常含 Webhook 模拟发送工具块。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 订阅树节点列表/树 | `GET /api/v1/alerts/subscriptions`、`GET /api/v1/alerts/subscriptions/tree`。 |
| 新建/编辑订阅节点 | `POST` / `PUT /api/v1/alerts/subscriptions/:id`；字段含匹配条件、静默窗口、接收组、恢复通知等。 |
| 删除/移动订阅节点 | `DELETE /api/v1/alerts/subscriptions/:id`、`POST /api/v1/alerts/subscriptions/:id/move`。 |
| 接收组管理 | `GET/POST/PUT/DELETE /api/v1/alerts/receiver-groups`。 |
| 历史事件 | `GET /api/v1/alerts/events`，支持集群、pipeline、group_key 等筛选。 |
| 统计 | `GET /api/v1/alerts/history/stats`。 |
| 模拟 Webhook | `POST` 类接口（以 `alert-channels` 或 alerts 服务实现为准，见前端调用）。 |

## 3. 数据表

- `alert_subscription_nodes`、`alert_receiver_groups`、`alert_events`。

## 4. 注意事项

- 订阅树匹配对 Alertmanager 来单与平台监控链路两条来源统一生效，注意 **monitor_pipeline**、`project_id` 等维度。  
- 节点表单与树编辑联动，删除父节点前需先处理子节点。
