# 需求说明：告警与监控平台

## 1. 目标

统一 **Prometheus 数据源**、**静默（含批量）**、**监控规则（PromQL）**、**告警策略**、**通知通道**、**处理人/部门**、**值班班次**；支持 Alertmanager Webhook 接入与历史事件查询。

## 2. 功能结构

```
告警通知 / 监控
├── 告警通道：钉钉/企邮/Webhook 等，字典项辅助密钥
├── 告警监控平台（Tab）
│   ├── 数据源
│   ├── 静默：单条编辑 + 从活跃告警批量创建
│   ├── 监控规则：数据源（已绑定项目）、PromQL、for/间隔/级别
│   ├── 处理人：用户 + 部门子树 + 恢复通知
│   ├── 值班：按规则维度的班次表，支持从其他规则复制班次
│   └── PromQL 查询 / 原生告警视图
├── 告警策略配置：匹配标签/正则、优先级、静默窗口、通道列表
└── 历史告警记录：事件列表与统计
```

## 3. 注意事项

| 项 | 说明 |
|----|------|
| 项目绑定 | 规则项目从数据源推导；通知邮箱 = 处理人 ∪ **项目启用成员邮箱** |
| 静默 | 时间区间、matcher 与 Alertmanager 语义对齐；批量静默对多条分别创建 |
| 值班 | `alert_duty_blocks` 挂 `monitor_rule_id`；与处理人邮箱合并去重 |
| 策略 | `match_labels_json` / `match_regex_json` 为 JSON 字符串，需合法 |

## 4. 相关表

`alert_channels`、`alert_policies`、`alert_datasources`、`alert_silences`、`alert_monitor_rules`、`alert_rule_assignees`、`alert_duty_blocks`、`alert_events`。
