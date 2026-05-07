# 告警订阅标签链路约定

本文约定 **订阅路由树**、**Prometheus 规则文件**、**平台监控规则（附加 Labels）** 三处如何对齐，避免 `match_labels_json`、匹配级别与各处 labels 重复或拼写不一致导致「无匹配通道」。

---

## 1. 云枢内部匹配顺序（实现语义）

对某一订阅树节点，是否命中按 **AND** 组合：

| 步骤 | 配置项 | 含义 |
|------|--------|------|
| ① | **匹配级别**（`match_severity`） | 与告警标签中的 **`severity`**（见下文）比较；**留空 = 不限制级别**。 |
| ② | **精确匹配**（`match_labels_json`） | 其中 **每一个键值对** 都必须与告警 `labels` **完全一致**（trim 后相等）。 |
| ③ | **正则匹配**（`match_regex_json`） | 可选；对对应键做正则匹配。 |

路由时使用的「级别」字符串来自告警 **`labels["severity"]`**（不是 `sevrity` 等别名）。若 **`severity` 缺失**，该字符串为空，此时若节点配置了匹配级别（如仅 `critical`），则 **无法命中**。

**项目维度**：订阅树根按 **`labels["project_id"]`**（字符串数字，如 `"1"`）选取；**缺少或无法解析时视为 0**，通常 **匹配不到任何根节点**。

---

## 2. 标准标签键（务必统一）

| 键 | 是否参与路由 | 说明 |
|----|----------------|------|
| **`project_id`** | **强烈必需** | 与平台项目主键一致，字符串形式（如 `"1"`）。决定进入哪棵订阅树。 |
| **`severity`** | **必需（若节点配置了匹配级别）** | 只允许 **`severity`**，禁止使用 `sevrity`、`servity`。取值如 `critical`、`warning`。 |
| **`cluster`** | 按节点配置 | 若在 `match_labels_json` 中写了 `cluster`，则告警侧必须带上相同值。 |
| **`route`** | 按节点配置 | 常用于区分钉钉/邮件等不同订阅分支；写了就必须一致。 |
| 其他业务键 | 按节点配置 | 出现在 `match_labels_json` / `match_regex_json` 中的键，告警侧都要有对应值。 |

**Prometheus `global.external_labels`**：适合放 **整环境通用** 且 **订阅树需要精确匹配** 的键（例如统一 `cluster`、`project_id`），减少每条规则重复写。

---

## 3. 订阅树节点：匹配级别 vs `match_labels_json` 怎么写

### 3.1 推荐写法（清晰、少重复）

- **匹配级别**：勾选本节点负责的级别（如 `critical`），用于「只看级别」的粗筛。
- **`match_labels_json`**：只写 **路由维度**，例如：
  ```json
  { "cluster": "prodK8s", "route": "prod-critical-dingding" }
  ```
- **不要在 JSON 里再写一遍 `severity`**（除非你需要「同级别但不同 JSON 组合」拆两条子路由）。级别交给 **匹配级别** 即可，避免与代码只认 `severity` 键不一致。

### 3.2 若必须在 JSON 里写 `severity`

则告警侧 **`labels["severity"]`** 必须与 JSON 中值一致；且仍建议 **匹配级别** 与之一致或留空其一，避免双重约束难以排查。

### 3.3 禁止事项

- JSON 中使用错误键名（如 `sevrity`）：告警携带的是 **`severity`** 时，**精确匹配永远失败**。
- `match_labels_json` 写得过多：每多一个键，**Prometheus / 平台规则** 就要多补一个标签，否则配不上。

---

## 4. Prometheus 规则文件（`rules/*.yml`）必须补什么

**最小集合（能进订阅树）：**

1. **`project_id`**：与目标项目一致（可与 `external_labels` 合并提供）。
2. **`severity`**：与节点匹配级别、模板一致。
3. **`match_labels_json` 中出现的每一个键`**：例如节点要求 `cluster`、`route`，则规则 `labels`（或 `external_labels`）必须提供相同键值。

示例（与节点 JSON `{ "cluster": "prodK8s", "route": "prod-critical-dingding" }` + 匹配级别 `critical` 对齐）：

```yaml
labels:
  severity: critical
  project_id: "1"
  cluster: prodK8s
  route: prod-critical-dingding
```

**Alertmanager**：`route.group_by`、`inhibit_rules` 里请使用与告警一致的键名（**`severity`**），否则分组与抑制行为与 Prometheus 标签不一致。

---

## 5. 平台监控规则：附加 Labels 必须补什么

平台在评估时会先构造一份基础 `labels`，再 **合并**「附加 Labels（JSON）」（后者覆盖同名字段）。

**平台默认会带的键（无需在附加 JSON 重复，除非你要覆盖）：**

- `alertname`、`severity`（来自规则表单）、`monitor_rule_id`、`datasource_id`、`project_id`（来自数据源所属项目）、`source`、`datasource_name`、`datasource_type`
- PromQL 向量第一个样本的 **metric 标签**（若与内置键不冲突则并入）

**你需要在「附加 Labels」中补充的，仅是订阅树要求、但平台不会自动生成的那部分**，典型为：

```json
{
  "cluster": "prodK8s",
  "route": "prod-critical-dingding"
}
```

若再在 JSON 里写 `severity`，会与表单级别合并覆盖——保持与 **匹配级别** 一致即可。

**注意**：内置监控规则依赖后端 **Redis** 做评估节拍与状态；Redis 不可用时规则不会进入 firing。

---

## 6. 端到端链路（简图）

```
Prometheus 规则 labels ∪ external_labels
        或
平台监控规则（内置 labels ∪ 附加 Labels JSON）
                    │
                    ▼  POST webhook（Alertmanager 路径）或平台内置 ReceiveAlertmanager
              labels["project_id"] → 选订阅树根
              labels["severity"]   → 节点「匹配级别」
              labels[*]            → match_labels_json / match_regex_json
                    │
                    ▼
              接收组 → 通道（钉钉等）→ alert_events
```

---

## 7. 排障清单（无匹配通道时）

1. **`project_id`** 是否在告警最终 `labels` 中且与项目一致？  
2. **`severity`** 键名是否拼写正确、值是否命中节点的匹配级别？  
3. **`match_labels_json` 每个键** 在告警 `labels` 中是否存在且值相等？  
4. 接收组是否绑定通道、`channel_ids_json` 非空且接收组在生效时间内？  
5. Prometheus 路径：Alertmanager 是否把告警送到云枢 webhook？（与「Prometheus 活跃告警」页面是否列表无关，后者直连 Prometheus。）

更通用的通知与聚合说明见：[告警通知与恢复通知使用说明](./alert-notify-guide.md)。

订阅树运维问答与「只有恢复」行为说明见：[告警路由与投递指南](./alert-routing-and-delivery-guide.md)。
