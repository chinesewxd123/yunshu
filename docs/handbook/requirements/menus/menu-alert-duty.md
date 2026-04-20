# 菜单需求：值班总览（`/alert-duty`）

## 1. 定位

- **路由**：`/alert-duty`，`alert-duty-page`。  
- **目标**：按 **项目 → 监控规则** 筛选，以**日/周**时间轴展示 **值班班次**（`alert_duty_blocks`），支持增删改班次；与「告警监控平台」内规则值班数据**同源**。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 选项目 | `GET /api/v1/projects`，再拉取该项目下监控规则（见 `listAlertMonitorRules` 封装）。 |
| 班次列表 | `GET /api/v1/alerts/duty-blocks?monitor_rule_id=...`（参数以前端 service 为准）。 |
| 新建/编辑班次 | `POST` / `PUT /api/v1/alerts/duty-blocks`、`/duty-blocks/:id`；含起止时间、用户、部门子树等。 |
| 删除 | `DELETE /api/v1/alerts/duty-blocks/:id`。 |
| 辅助数据 | `getUsers`、`getDepartmentTree` 与监控平台处理人页一致。 |

## 3. 数据表

- `alert_duty_blocks`，外键语义为 `monitor_rule_id`。

## 4. 注意事项

- 与 **规则处理人**邮箱在告警发送链路中**合并去重**；勿与监控平台重复配置冲突。  
- 时间轴为可视化辅助，以数据库起止时间为准。
