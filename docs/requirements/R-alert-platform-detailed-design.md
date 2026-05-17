# 告警平台需求与设计说明（与实现对齐）

本文描述 **yunshu** 当前告警与监控平台的后端行为、数据模型、HTTP API、关键算法与配置映射，**与仓库代码一致**，便于评审与二次开发。配套的运维与标签约定见：

- [告警路由与投递指南](../alert-routing-and-delivery-guide.md)
- [告警订阅标签链路约定](../alert-subscription-labels-chain.md)
- [告警通知与恢复通知使用说明](../alert-notify-guide.md)

---

## 1. 目标与范围

| 能力 | 说明 |
|------|------|
| Webhook 接入 | Alertmanager `POST /api/v1/alerts/webhook/alertmanager`，载荷形态兼容 Alertmanager |
| 内置监控规则 | 平台配置 PromQL + `for`，后端定时查询 Prometheus，**不经过 Alertmanager** |
| 订阅树路由 | 弃用旧策略表路径，**仅**订阅树 → 接收组 → 通道 |
| 静默 / 抑制 | 平台静默表；告警抑制服务（可选） |
| 渠道投递 | 钉钉、企业微信、邮件、通用 Webhook 等 |
| 审计 | `alert_events` 记录抑制原因、通道结果 |
| 值班 / 处理人 | 监控规则维度：`alert_rule_assignees`、`alert_duty_blocks` |

---

## 2. 架构与两条告警入口

### 2.1 入口 A：Prometheus → Alertmanager → 云枢

1. Prometheus 规则评估，`for` 满足后进入 firing，推送 Alertmanager。  
2. Alertmanager 路由到 webhook，`POST` 云枢（鉴权见 `alert.webhook_token`）。  
3. `AlertHandler.ReceiveAlertmanager` → `AlertService.ReceiveAlertmanager`。

**路由注册**：`internal/router/router.go` — `alertWebhook.POST("/webhook/alertmanager", ...)`（无通用 auth，token 校验在 handler）。

### 2.2 入口 B：平台监控规则（内置）

1. `AlertService.runMonitorRuleEvaluator`：约 **5s** tick → `tickMonitorRules`。  
2. `evaluateOneMonitorRule`：`promapi.Client.QueryInstant` 调数据源 Prometheus。  
3. `evaluateMonitorRuleWithRedis`：pending/`for_seconds` 状态在 Redis；触发后 **`ReceiveAlertmanager`**，`Receiver` 固定为 **`platform-monitor`**。

**约束**：`tickMonitorRules` 仅在 **`s.redis != nil`** 时执行单条规则评估（见 `alert_monitor_evaluator.go`）。无 Redis 时启用规则不会运行。

### 2.3 入口 C（扩展）：云资源到期等

`alert_cloud_expiry_evaluator.go` 等评估路径同样汇聚到 `ReceiveAlertmanager`，`receiver` 特定字符串（如 `cloud-expiry`），`resolveAlertDatasourceMeta` 映射流水线名称。

---

## 3. 核心代码索引

| 模块 | 路径 | 职责 |
|------|------|------|
| HTTP Webhook / 通道 CRUD / 事件列表 | `internal/handler/alert_handler.go`、`alert_platform_handler.go`、`alert_subscription_handler.go`、`alert_receiver_group_handler.go` | 入参绑定、调用 Service |
| 统一投递 | `internal/service/alert_service.go` | `ReceiveAlertmanager`、`channelIDSetForAlert`、`sendToChannel` |
| 分组节流 | `internal/service/alert_aggregate_state.go` | `decideFiringGroupTiming`、`markResolvedNotificationSent`、`markAlertFiringDelivered` |
| 订阅树 | `internal/service/alert_subscription_service.go` | `MatchRouteDetailed`、`nodeMatches` |
| 平台规则评估 | `internal/service/alert_monitor_evaluator.go`、`alert_monitor_redis.go` | PromQL、Redis for |
| 渠道渲染与发送 | `internal/service/alert_delivery.go` | `sendToChannel`、`mergeAssigneeEmails`（邮件处理人优先） |
| 配置 | `internal/config/config.go` → `AlertConfig` | Webhook token、group_*、Prometheus 增强 |

---

## 4. 数据模型（GORM）

| 表 / 模型文件 | 用途 |
|---------------|------|
| `alert_channels` | `alert_channel.go` — 通道类型、URL、Headers JSON（含模板与匹配） |
| `alert_events` | `alert_event.go` — 历史记录、抑制原因、`matched_policy_*` |
| `alert_datasources` | `alert_datasource.go` — Prometheus 等数据源，绑定项目 |
| `alert_monitor_rules` | `alert_monitor_rule.go` — PromQL、for、labels JSON |
| `alert_silences` | `alert_silence.go` — 平台静默 |
| `alert_subscription_nodes` | `alert_subscription.go` — 订阅树节点 |
| `alert_receiver_groups` | `alert_subscription.go` — 接收组 → `channel_ids_json` |
| `alert_rule_assignees` | `alert_rule_assignee.go` — 规则处理人 |
| `alert_duty_blocks` | `alert_duty_block.go` — 值班班次，挂 `monitor_rule_id` |
| `alert_inhibition_*` | `alert_inhibition.go` — 抑制规则（若启用） |

---

## 5. HTTP API 清单（`/api/v1` 前缀）

摘自 `internal/router/router.go`：

| 方法与路径 | 说明 |
|------------|------|
| `POST /alerts/webhook/alertmanager` | Alertmanager Webhook（token） |
| `GET/POST/PUT/DELETE .../alerts/channels*` | 告警通道 |
| `GET /alerts/events`、`GET /alerts/history/stats` | 事件与统计 |
| `GET/POST/PUT/DELETE .../alerts/datasources*` | 数据源；`GET .../:id/ping` 连通性（PromQL `vector(1)`）；`GET .../prometheus-alerts` 活跃告警；`POST .../query` PromQL |
| `GET/POST/PUT/DELETE .../alerts/silences*` | 静默 |
| `GET/POST/PUT/DELETE .../alerts/monitor-rules*` | 监控规则；`.../assignees` |
| `GET/POST/PUT/DELETE .../alerts/duty-blocks` | 值班 |
| `GET/POST/PUT/DELETE .../alerts/subscriptions*` | 订阅树 |
| `GET/POST/PUT/DELETE .../alerts/receiver-groups` | 接收组 |
| `GET/POST/PUT/DELETE .../alerts/cloud-expiry-rules*` | 云到期规则 |

---

## 6. `ReceiveAlertmanager` 处理流水线（逻辑顺序）

实现位置：`internal/service/alert_service.go` — `ReceiveAlertmanager`。

对 `payload.Alerts` 中每一条：

1. **合并标签**：`mergeStringMap(CommonLabels, alert.Labels)`；`resolveAlertDatasourceMeta` 写入 `datasource_*`、`monitor_pipeline`。  
2. **规范化**：`eventName`、`severity`（缺省 `warning`）、`status`（缺省 `firing`）。  
3. **维度**：`alertnotify.ExtractDims`、`computeGroupKey`、`DigestLabels`、`resolveAlertEnvironmentLabel`。  
4. **平台静默**：命中则 `logSilenceSuppressed`，`continue`。  
5. **指纹计数**：`updateFingerprintState`。  
6. **抑制**：firing 时 `CheckInhibition`，可能 `continue`。  
7. **组装 outgoing**：摘要、`current`（缓存/Prometheus）、`enrichOutgoingProjectName`、`enrichAssigneeAndDutyEmails`（基于 `monitor_rule_id`）。  
8. **firing 分组节流**：`decideFiringGroupTiming`；不通过则 `logSuppressedFiringTiming`，`continue`。  
9. **resolved 去重占位**：在通过订阅匹配等检查之后、`markResolvedNotificationSent`（避免过早占位）。  
10. **订阅路由**：`channelIDSetForAlert` → `MatchRouteDetailed(project_id, labels, severity, status)`；无通道则 `logNoMatchedChannel`。  
11. **订阅静默**：`shouldSuppressByRouteSilence`。  
12. **resolved 无 prior firing**：若 Redis 记录从未成功 firing 投递，则 `logResolvedSuppressedNoPriorFiringDelivery` 并跳过外发。  
13. **通道循环**：`sendToChannel`；firing 至少一个 HTTP 2xx → `markAlertFiringDelivered`。  
14. **失败审计**：`sentCount>0` 且全部非 2xx → `logAllChannelsDeliveryFailed`；resolved 外发失败可 `clearResolvedNotificationSent`。  
15. **resolved 收尾**：清理 fingerprint、group timing、`clearAlertFiringDelivered`。

---

## 7. 订阅树匹配规则（实现）

`internal/service/alert_subscription_service.go`：

- 按 **`labels["project_id"]`** 选根节点集合。  
- **匹配级别**（`match_severity`）：与 **`labels["severity"]`** 比较（配置为空则不过滤）。  
- **`match_labels_json`**：逐项精确相等。  
- **`match_regex_json`**：正则匹配。  
- **接收组**：命中节点合并 `ReceiverGroupIDs`，去重；接收组需在生效时间内且 `channel_ids_json` 解析出通道 ID。

路由入口：`channelIDSetForAlert` **仅**使用订阅树（注释载明弃用旧策略）。

---

## 8. 配置项映射（`configs/config.yaml` → `AlertConfig`）

| 配置键 | 用途 |
|--------|------|
| `webhook_token` | Alertmanager Webhook 校验 |
| `group_wait_seconds` / `group_interval_seconds` / `repeat_interval_seconds` | `decideFiringGroupTiming`（Redis） |
| `dedup_ttl_seconds` | 指纹与 resolved 标记 TTL |
| `aggregate_ttl_seconds` | 分组 timing Redis key TTL |
| `prometheus_url` + enrich 相关 | 异步查询当前值 |
| `platform_limits.*` | 钉钉/企微/通用 Webhook 正文长度 |

---

## 9. Redis Key 约定（节选）

| Key 模式 | 含义 |
|----------|------|
| `alert:group:timing:{groupKey}` | firing 分组节流状态 |
| `alert:firing_delivered:{fingerprint}` | 已成功 firing 通道投递（HTTP 2xx） |
| `alert:resolved:sent:{fingerprint}` | resolved 已尝试发送占位（失败可清除） |
| `alert:mon:state:{ruleID}` | 平台监控规则评估状态 |
| `alert:subscription:silence:*` | 订阅节点静默窗口 |

---

## 10. 处理人、值班与通知标题

- **处理人邮件**：`ResolveNotifyEmailsDirectUsers` — 仅显式用户 + `extra_emails`；部门子树不参与邮件。  
- **处理人 IM**：`ResolveNotifyPhones` — 用户 + 部门（项目成员 ∩ 子树 + 负责人）；部门需 UI 手动配置，保存后按库加载。  
- `enrichAssigneeAndDutyEmails`：合并处理人直发邮箱与**当前时刻**值班班次邮箱/手机 → `assignee_emails` / `assignee_phones`。  
- `mergeAssigneeEmails`：**邮件通道**若 `assignee_emails` 非空，仅发往这些地址；且不合并接收组静态抄送。  
- **值班标题**：`HasActiveBlockAtRule` 为真时，`buildUnifiedNotifyTitle` 前缀 `值班`。  
- 钉钉/企微：手机号解析失败等场景可补邮件（`alert_delivery_assignee_expand.go`）。  
- **规则列表**：`GET /alerts/monitor-rules?enabled=true|false`；`enabled=false` 不参与 `tickMonitorRules`。

---

## 11. 非功能需求（来自实现）

- Webhook 与部分告警路径依赖 **Redis**；平台监控规则强依赖 Redis。  
- 单进程内 Kom/K8s 与本告警模块独立，无耦合。  
- 操作审计：告警 API 路由组使用 `opAudit` 中间件（见 router）。

---

## 12. 数据源连通性检测

| 项 | 说明 |
|----|------|
| API | `GET /api/v1/alerts/datasources/:id/ping` |
| 实现 | `AlertDatasourceService.PingDatasource`：复用 `promapi.Client.QueryInstant(ctx, "vector(1)", "")`，与 `PromQuery` 同源 TLS/鉴权；使用 `getRaw` **不**要求数据源处于启用状态，便于停用期间排查 |
| 响应 | `DatasourcePingResult`：`ok`、`message`、`latency_ms`（失败时仍返回 JSON，`ok=false`） |
| PromQL | `internal/pkg/promapi.QueryResponseStatusSuccess` 校验返回 JSON 的 `status == success` |

---

## 13. 已知架构风险与改进方向（按评审结论对齐）

以下条目与产品/架构评审描述 **一一对应**：先写 **风险点（当前实现的问题）**，再写 **改进点（目标方向）**。  
**说明**：本节为需求与设计层面的整改清单；**不代表当前代码已全部实现**，落地需在迭代中单独立项开发与验收。

### 13.1 架构层面

#### （1）强行复用 Alertmanager Payload 作为内部统一模型

| 维度 | 内容 |
|------|------|
| **风险点** | 所有告警入口（Alertmanager Webhook、平台内置规则、`cloud-expiry` 等）均将自身数据适配为 Alertmanager Webhook 形态后汇入 `ReceiveAlertmanager`。由此带来：**侵入式适配**——新增告警源须迁就 AM 字段集；**难以原生扩展**云资源类字段（到期时间、资源 ID 等）；**语义被掩盖**——内置规则的 `for` 状态、云告警到期逻辑等与 AM 告警生命周期混在同一结构中，维护成本高。 |
| **改进点** | 定义 **平台内部统一的告警核心模型（canonical alert）**；各告警源仅负责转换为该模型；统一处理流水线（静默、路由、投递等）只消费内部模型；对外再按需适配 Alertmanager 或其它 Northbound API。 |

#### （2）Webhook 同步处理整条流水线

| 维度 | 内容 |
|------|------|
| **风险点** | Alertmanager Webhook 请求在当前实现中为 **同步**：须完成鉴权、静默、抑制、订阅匹配、通道发送等后才返回。大批量告警或单个慢通道（如邮件）会拉长响应时间，易触发 **AM 客户端超时与重试**，进而带来 **重复处理与重复通知**。 |
| **改进点** | Webhook **快速路径**：完成鉴权与参数校验后，将告警写入 **内存队列或 Redis 队列**，**立即返回 HTTP 200（或 202）**；由后台 **Worker** 异步执行后续流水线，从根上规避 AM 因超时导致的重试风暴。 |

#### （3）定时评估任务缺少集群级互斥

| 维度 | 内容 |
|------|------|
| **风险点** | 平台内置规则评估（`tickMonitorRules`）本质为进程内定时任务；**多实例部署**时若缺乏可靠的集群级互斥，可能出现多副本 **并行全量评估**，导致重复查询 Prometheus、重复触发与重复通知，浪费资源且干扰运维判断。 |
| **改进点** | 基于 **Redis（或其它分布式协调组件）实现分布式锁 / Leader 选举**，保证 **同一时刻仅一个实例** 执行一轮全局规则调度（或明确「按 rule 分片」策略并在文档中固化语义）。 |

### 13.2 可靠性与可用性层面

#### （4）强依赖 Redis 且无明确降级

| 维度 | 内容 |
|------|------|
| **风险点** | 规则评估节拍与状态、`decideFiringGroupTiming`、`firing_delivered` 标记等多处 **强依赖 Redis**；文档亦写明无 Redis 时部分能力直接不可用。Redis 宕机、重启或网络抖动时，易出现 **内置规则停摆、节流与状态管理失效**，平台可用性过度绑定单一组件。 |
| **改进点** | **核心状态双写或兜底**：例如 Redis + 数据库，Redis 不可用时 **降级读库**；**非核心能力可关闭**：节流、订阅静默窗口等在 Redis 不可用时 **自动跳过**，保证 **告警仍能投递**；对外明确 **SLA 与降级行为**（监控 Redis、自动切换策略）。 |

#### （5）恢复通知与 `firing_delivered` 生命周期

| 维度 | 内容 |
|------|------|
| **风险点** | 恢复路径依赖 Redis 中的 **`firing_delivered`** 等标记；若标记 **TTL（如与 `dedup_ttl_seconds` 相关）短于告警持续时长**，可能出现 **长时间 firing 后恢复时标记已过期**，用户 **收不到恢复通知**，形成「只知故障不知恢复」的体验断裂。 |
| **改进点** | **告警触发时在数据库持久化「已成功 firing 投递」事实**（或等价审计记录）；Redis 标记仅作 **加速缓存**；恢复发送决策以 **数据库记录为最终依据**，避免因 Redis TTL 丢失而导致恢复被错误抑制。 |

### 13.3 其它补充观察（次要，待后续评审是否纳入正式改进项）

| 观察 | 代码/行为线索 | 备注 |
|------|----------------|------|
| `groupKey` 为空时节流旁路 | `decideFiringGroupTiming` | 部分告警可能完全不经过分组节流 |
| 订阅树缓存刷新间隔 | `AlertSubscriptionService` ~30s | 配置变更后短时间仍可能旧路由 |
| 平台规则指纹 | `monitor_rule_{id}` | 与 Prometheus fingerprint 语义不同 |
| Webhook 幂等 | AM 重试 | 极端场景可能多条 `alert_events` |

---

## 14. 变更记录（文档维护）

| 日期 | 说明 |
|------|------|
| 2026-05-08 | 初版：与当前 `ReceiveAlertmanager`、订阅树、聚合、firing/resolved 语义对齐 |
| 2026-05-08 | 增加数据源 `ping` API、§13 风险与改进（按评审条目对齐） |
