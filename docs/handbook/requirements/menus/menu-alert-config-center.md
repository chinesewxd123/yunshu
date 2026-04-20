# 菜单需求：告警配置中心（`/alert-config-center`）

## 1. 定位

- **路由**：`/alert-config-center`，`alert-config-center-page`（可由动态菜单或书签进入）。  
- **目标**：**告警策略**的增删改查与 **历史告警事件**检索；顶部常含 Webhook 模拟发送工具块。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 策略列表 | `GET /api/v1/alerts/policies`。 |
| 新建/编辑策略 | `POST` / `PUT /api/v1/alerts/policies/:id`；字段含优先级、静默窗口、通道 ID 列表、`match_labels_json`、`match_regex_json` 等。 |
| 删除 | `DELETE /api/v1/alerts/policies/:id`。 |
| 历史事件 | `GET /api/v1/alerts/events`，支持集群、pipeline、group_key 等筛选。 |
| 统计 | `GET /api/v1/alerts/history/stats`。 |
| 模拟 Webhook | `POST` 类接口（以 `alert-channels` 或 alerts 服务实现为准，见前端调用）。 |

## 3. 数据表

- `alert_policies`、`alert_events`。

## 4. 注意事项

- 策略匹配 Alertmanager 来单与平台监控链路两条来源时，注意 **monitor_pipeline** 等维度区分。  
- 表单已使用**右侧 Drawer**，避免遮挡宽表。
