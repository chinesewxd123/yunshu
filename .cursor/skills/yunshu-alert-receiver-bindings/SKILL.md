---
name: yunshu-alert-receiver-bindings
description: Diagnoses Yunshu (yunshu) alert delivery when subscription tree matches but no notification fires; maps migrated receiver group names to alert_channels IDs and gives verification SQL. Use when debugging alert_receiver_groups channel_ids_json, subscription nodes receiver_group_ids_json, Alertmanager webhook paths, or no_policy_matched / no matched channel in alert_events.
disable-model-invocation: true
---

# Yunshu 告警：订阅树命中却无通知

## 核心结论（先查这三类）

1. **订阅节点 id ≠ 接收组 id**  
   `alert_subscription_nodes.id` 与 `alert_receiver_groups.id` 无对应关系。必须以节点的 **`receiver_group_ids_json`** 为准查 **`alert_receiver_groups.id`**。

2. **`channel_ids_json` 为空则必然无通道**  
   路由命中后仍会展开接收组得到 0 个 `alert_channels`，历史上记 `error_message=no_policy_matched` 且通道列为「无匹配通道」（实现上 matched 但集合为空也会走到同一 reason）。

3. **`labels.project_id` 决定匹配哪棵树**  
   根节点按 `project_id` 建索引；Webhook 告警若无 `project_id` 会按 0 处理，可能与 UI 上项目 1 的树不一致。

## 验证 SQL（按需执行）

```sql
-- 节点绑了哪个接收组
SELECT id, name, project_id, receiver_group_ids_json
FROM alert_subscription_nodes
WHERE deleted_at IS NULL AND id = ?;

-- 接收组是否绑了通道
SELECT id, name, project_id, channel_ids_json, enabled
FROM alert_receiver_groups
WHERE deleted_at IS NULL AND id IN (...);

-- 通道是否存在且启用
SELECT id, name, type, enabled FROM alert_channels WHERE deleted_at IS NULL ORDER BY id;
```

## 迁移组名与通道约定（与典型 dump 一致时）

默认 `alert_channels`：**1=企微 wechat_work，2=钉钉 dingding，3=邮件 email**。  
`migrated:*` 接收组建议绑定：

| name | channel_ids_json |
|------|------------------|
| migrated:prod-warning-email | `[3]` |
| migrated:prod-critical-dingding | `[2]` |
| migrated:prod-critical-all | `[3,2,1]` |
| migrated:prod-warning-wecom-email | `[1,3]` |
| migrated:prod-info-email | `[3]` |

若线上 channel id 不同，先 `SELECT id,type FROM alert_channels` 再改 JSON。更新后重启进程或等待接收组缓存刷新。

## Alertmanager Webhook

- 路径：**`POST /api/v1/alerts/webhook/alertmanager`**（GET 会 404）。  
- `connection refused` 为 TCP 层失败，与 token 是否正确无关。  
- 平台侧另有 firing **分组节流**（如 `repeat_suppressed`），短时间重复相同 group_key 可能不再外发。

## 不要做的事

- 不要用订阅节点 id 去查 `alert_receiver_groups WHERE id = 同一数字`。  
- 不要假设 UI 里选了「钉钉策略」就等于接收组内已配置钉钉通道；通道在**接收组编辑**或库里 `channel_ids_json`。
