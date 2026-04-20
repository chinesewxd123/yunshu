# 菜单需求：告警监控平台（`/alert-monitor-platform`）

## 1. 定位

- **路由**：`/alert-monitor-platform`，`alert-monitor-platform-page`。  
- **目标**：一站式完成 **Prometheus 数据源**、**静默**、**监控规则（PromQL）**、**规则处理人**、**值班班次**、**PromQL/原生告警查询**；内嵌 **告警配置中心**（策略与历史）子面板。

## 2. Tab 与功能对应

| Tab | 核心能力 | 主要 API（前缀 `/api/v1/alerts`） |
|-----|----------|-----------------------------------|
| 数据源 | CRUD、连通、即时/范围查询 | `/datasources`、`/datasources/:id/query(_range)`、`prometheus-alerts` |
| 静默 | 单条创建/编辑、从活跃告警批量静默 | `/silences`、`/silences/batch` |
| 监控规则 | PromQL、for、间隔、级别、绑定项目 | `/monitor-rules`、处理人 `/monitor-rules/:id/assignees`、值班 `/duty-blocks` |
| 策略与历史 | 嵌入 `AlertConfigCenterPanel` | `/policies`、`/events`、`/history/stats` |
| PromQL 查询 | 调试 PromQL | 同数据源 query 接口 |

## 3. 业务规则（必读）

- **项目绑定**：监控规则所属项目由所选数据源派生；告警通知邮箱 = **处理人解析结果 ∪ 项目启用成员邮箱**（去重）。  
- **静默**：匹配器语义对齐 Prometheus/Alertmanager；批量静默对多条告警分别创建静默记录。  
- **值班**：`alert_duty_blocks` 按 `monitor_rule_id` 挂接；可复制其他规则下班次到当前规则（新建独立记录）。  
- **UI**：新建/编辑类表单已统一为**右侧 Drawer**，便于对照列表与说明文案。

## 4. 数据表

- `alert_datasources`、`alert_silences`、`alert_monitor_rules`、`alert_rule_assignees`、`alert_duty_blocks`；策略/事件见告警配置文档。

## 5. 权限

- 各子接口在 `defaultPermissions` 中拆分为多条；需对业务角色显式授权。

## 6. 注意事项

- PromQL 与阈值单位、条件构建器并用时，以最终写入规则的 `expr` 为准。  
- 数据源 `base_url`、Basic 等可结合**数据字典**自动完成（见页面提示）。
