# 告警通知与恢复通知使用说明

本文档说明 `yunshu` 当前已实现的告警能力，覆盖从配置到发送、从聚合到排障的完整流程。

## 1. 总体流程

1. Alertmanager 将告警发送到后端接口：`POST /api/v1/alerts/webhook/alertmanager`
2. 后端解析 `labels/annotations`，提取维度（如 `cluster/namespace/workload`）
3. 生成统一消息模板（钉钉/企业微信/通用 webhook）
4. 按规则进行聚合收敛（firing 与 resolved 分开）
5. 按渠道发送，并按平台限制进行分段/裁剪
6. 发送结果落库到 `alert_events`，用于审计与查询

---

## 2. 后端配置（`configs/config.yaml`）

告警配置在 `alert` 段：

```yaml
alert:
  webhook_token: "change-me-alert-token"
  default_timeout_ms: 5000
  max_payload_chars: 8000
  dedup_ttl_seconds: 86400

  group_by: ["alertname", "cluster", "namespace", "severity", "receiver"]
  group_wait_seconds: 0
  group_interval_seconds: 60
  repeat_interval_seconds: 300
  aggregate_ttl_seconds: 86400

  platform_limits:
    dingding_max_chars: 4500
    wecom_max_chars: 3500
    generic_max_chars: 8000

  prometheus_url: "http://127.0.0.1:9090"
  prometheus_token: ""
  prom_query_timeout_seconds: 5
  prometheus_enrich_enabled: true
  prometheus_enrich_queue_size: 1024
  prometheus_enrich_workers: 4
```

### 2.1 参数说明

- `webhook_token`：Alertmanager webhook 鉴权 token（请求头 `X-Alert-Token` 或 query `token`）
- `group_by`：服务端 `group_key` 的计算维度
- `group_wait_seconds`：首次见到某个 `group_key` 后的等待窗口（收集同组告警），到达后才允许“首次发送”
- `group_interval_seconds`：已发送后，若组内容发生变化（平台用 `labels_digest` 近似）再次发送的最小间隔
- `repeat_interval_seconds`：持续 firing 且无新变化时的重复提醒间隔
- `aggregate_ttl_seconds`：聚合状态在 Redis 的保留时间
- `platform_limits.*`：平台消息预算（当前按请求体 JSON 大小限制进行控制）
- `prometheus_url`：用于从 `generatorURL` 解析表达式并查询当前值
- `prometheus_enrich_enabled`：开启 P3 异步增强（通知先发，Prometheus 查询后台执行）
- `prometheus_enrich_queue_size`：异步增强队列长度（满时丢弃任务，不阻塞主链路）
- `prometheus_enrich_workers`：后台 worker 并发数

---

## 3. Alertmanager 对接配置

推荐将 Alertmanager 作为第一层聚合，当前服务作为第二层防抖与多渠道分发。

示例：

```yaml
route:
  receiver: "yunshu-webhook"
  group_by: ['alertname','cluster','namespace','severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 2h

receivers:
- name: "yunshu-webhook"
  webhook_configs:
  - url: "http://<host>:8080/api/v1/alerts/webhook/alertmanager?token=change-me-alert-token"
    send_resolved: true
```

建议在 Prometheus 或 Alertmanager 中统一提供 `cluster` 标签（例如 external_labels）：

```yaml
global:
  external_labels:
    cluster: prod-1
```

**项目名称（通知标题第三段）**：与平台「项目」表主键一致时，在规则 **labels** 中增加 `project_id`（字符串数字即可，如 `"1"`）。后端会按 ID 查询 `projects.name` 填入标题与 `project_name` 字段。也可直接写 `project_name` 标签覆盖显示名（不查库）。

单条 Prometheus 告警规则示例：

```yaml
groups:
  - name: example
    rules:
      - alert: HostDisk
        expr: disk_used_percent > 80
        labels:
          severity: warning
          project_id: "1"
        annotations:
          summary: "磁盘使用率过高"
```

**订阅树与标签链路**（`match_labels_json`、匹配级别、Prometheus / 平台规则 labels 如何对齐）：见 [告警订阅标签链路约定](./alert-subscription-labels-chain.md)。

**路由、值班、单节点、resolved 行为等运维说明**：见 [告警路由与投递指南](./alert-routing-and-delivery-guide.md)。

---

## 4. 通道配置（页面：Webhook 告警通道）

页面支持通道类型：
- `dingding`
- `wechat_work`（企业微信）
- `generic_webhook`
- `email`

### 4.1 钉钉机器人

- `Webhook URL`：填原始机器人地址（含 access_token）
- `signSecret`：填“加签密钥”（如开启加签）
- 系统会自动追加 `timestamp/sign`，无需手工拼接

### 4.2 企业微信机器人/应用

- 机器人模式：支持 `atMobiles/atUserIds/isAtAll`
- 应用模式：需配置 `corpID/corpSecret/agentId`

### 4.3 轻量路由匹配（按通道）

可在 `headers_json` 增加匹配规则，仅命中时发送：

```json
{
  "matchLabels": { "cluster": "prod-1" },
  "matchRegex": { "namespace": "^(kube-system|monitoring)$" }
}
```

说明：
- `matchLabels`：精确匹配
- `matchRegex`：正则匹配（正则非法会视为不匹配，避免误发）

---

## 5. 告警与恢复的聚合收敛规则（对齐夜莺/Alertmanager）

## 5.1 firing（告警触发）

- 以 `group_key` 为维度执行 group 时序语义：
  - `group_wait_seconds`：首次见到 group 后先等待（收集同组告警），窗口内写抑制事件（`group_wait_suppressed`）
  - `group_interval_seconds`：已发送后若组内发生“新变化”（平台用 `labels_digest` 近似），在窗口内写抑制事件（`group_interval_suppressed`）
  - `repeat_interval_seconds`：持续 firing 且无变化时，在窗口内写抑制事件（`repeat_suppressed`）

## 5.2 resolved（告警恢复）

- 已支持恢复通知（依赖 Alertmanager `send_resolved: true`）
- 对齐夜莺语义：同一告警实例（同 `fingerprint`）的恢复通知 **仅发送一次**；
  - 重复 resolved 事件会写库（`resolved_aggregate_suppressed`）但不会重复外发

恢复汇总示例摘要：

- `恢复汇总：N 条在 t1 ~ t2 已恢复`

---

## 6. 消息模板与长度控制

### 6.0 P3 异步增强（云原生推荐）

当前已采用 P3 思路：

1. 告警通知主链路不再同步阻塞 Prometheus 查询
2. `current` 字段优先读取 Redis 缓存
3. 后台 worker 异步查询 Prometheus 并刷新缓存
4. 队列满时丢弃异步查询任务，确保通知链路稳定

说明：
- `current` 语义是“最近一次异步刷新值（发送时缓存）”
- 不作为触发依据，只用于增强可读性

## 6.1 统一模板字段

当前模板包含核心字段：
- 状态/级别
- `cluster/namespace/workload/pod/node/instance`
- 摘要（summary/description）
- 当前值（从 Prometheus 查询）
- 时间、接收器、指纹、链接

## 6.2 分段策略（防超限）

为降低平台拒收概率，当前采用：

1. 先构建最终 body
2. 按 JSON 请求体大小判断是否超限
3. 超限先做内容降噪：
   - 优先删除“元信息”
   - 再裁剪“摘要”
4. 仍超限则分段发送（UTF-8 安全切分）

说明：
- 分段适用于钉钉/企业微信
- 通用 webhook 使用 `generic_max_chars` 兜底裁剪

---

## 7. 事件记录与查询

每次发送（或抑制）都会写 `alert_events`，关键字段：

- `title/source/severity/status`
- `cluster/group_key/labels_digest`
- `channel_name/success/http_status_code`
- `error_message/request_payload/response_payload`

页面 `Webhook 告警事件` 支持按以下条件筛选：
- `keyword`
- `cluster`
- `group_key`

---

## 8. 验证清单（上线前建议）

1. 用测试通道发送一条 `firing`：确认模板、@、链接与时间正确
2. 连续发送同 `group_key` firing：确认仅首条或按间隔发送
3. 连续发送同 `group_key` resolved：确认短窗口合并为恢复汇总
4. 构造超长摘要：确认不会超限，且可读性可接受
5. 配置 `matchLabels/matchRegex`：确认仅目标通道接收
6. 查看 `alert_events`：确认成功/失败/抑制原因可追踪

---

## 9. 常见问题

### 9.1 为什么没收到恢复通知？

- Alertmanager 是否设置了 `send_resolved: true`
- webhook token 是否匹配
- 是否被路由匹配规则过滤
- 是否在恢复去抖窗口内被合并（查看 `alert_events` 中 `resolved_aggregate_suppressed`）

### 9.2 为什么消息被拆成多条？

- 触发了平台长度限制保护（正常行为）
- 可适当调大 `platform_limits`，但建议保守，避免平台拒收

### 9.3 钉钉 URL 是否需要手动加 timestamp/sign？

- 不需要。填原始机器人 URL 与 `signSecret`，系统自动追加签名参数

### 9.4 `curl .../api/v1/rules` 返回 `groups: []`，收不到任何告警

说明 **Prometheus 未加载任何告警规则**。常见原因：

- `rule_files` 指向的目录不存在、为空，或 YAML 语法错误
- 修改 `prometheus.yml` / 规则文件后 **未 reload / 未重启** Prometheus 进程
- 实际运行的 Prometheus **不是**你以为的那份配置（例如 systemd 用的路径与手动 `cat` 不一致）

处理：在规则目录放置至少一条可触发的 `groups` 规则，执行 reload 后再次检查：

```bash
curl -s "http://127.0.0.1:9091/api/v1/rules"
curl -s "http://127.0.0.1:9091/api/v1/alerts"
```

### 9.5 「Event 事件」菜单点进去却是 Webhook 告警事件

多为 **菜单 path/component 在库中被误改**。服务端在加载菜单树时会自动纠正：

- 名称为「Event 事件」且 path 误为 `/alert-events` 时，改回 `/events`，`component` 为 `events-page`
- path 为 `/events` 时，确保 `component` 为 `events-page`，并将图标与「Webhook 告警通道」区分（`FileSearchOutlined`）

刷新页面或重新登录后生效；若仍异常，可在「菜单管理」中手动核对 **路径** 与 **组件**。

---

## 10. 生产建议（规范实践）

- 第一层聚合收敛尽量在 Alertmanager 做（`group_by/group_interval/repeat_interval/inhibit_rules/silence`）
- 本服务的聚合用于“第二层保护”，防止告警风暴刷屏
- 固定使用 `cluster` 作为多集群主维度，并统一标签命名
- 定期清理无效告警规则，避免“长期噪音告警”

