# 菜单需求：值班总览（`/alert-duty`）

## 1. 定位

- **路由**：`/alert-duty`，`alert-duty-page`。  
- **目标**：按 **项目 → 监控规则** 筛选，以**日/周**时间轴展示 **值班班次**（`alert_duty_blocks`），支持增删改班次；与「告警监控平台」内规则值班数据**同源**。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 选项目 | `GET /api/v1/projects`，再拉取该项目下监控规则（见 `listAlertMonitorRules` 封装）。 |
| 班次列表 | `GET /api/v1/alerts/duty-blocks?monitor_rule_id=&project_id=`；表格操作列 **fixed: right**。 |
| 新建/编辑班次 | `POST` / `PUT /api/v1/alerts/duty-blocks`、`/duty-blocks/:id`；含起止时间、用户、部门子树等。 |
| 删除 | `DELETE /api/v1/alerts/duty-blocks/:id`。 |
| 辅助数据 | `getUsers`、`getDepartmentTree` 与监控平台处理人页一致。 |

## 3. 数据表

- `alert_duty_blocks`，外键语义为 `monitor_rule_id`。

## 4. 注意事项

- 与 **规则处理人**直发邮箱在告警发送链路中**合并去重**；部门子树仅影响 IM @，不向部门全员发邮件。  
- **当前时刻**命中班次时，外发通知标题前缀 **`值班`**（见 `buildUnifiedNotifyTitle`）。  
- 时间轴为可视化辅助，以数据库 `starts_at` / `ends_at` 为准。  
- 部门需手动选择，**不会**随值班人勾选自动回填（与监控平台处理人抽屉一致）。
