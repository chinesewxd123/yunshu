# 菜单需求：Webhook 告警通道（`/alert-channels`）

## 1. 定位

- **路由**：`/alert-channels`，组件 `alert-channels-page`。  
- **所属分组**：告警通知。  
- **目标**：维护钉钉、企业微信、邮件、Webhook 等**通知通道**，供告警策略与 Alertmanager 转发使用。

## 2. 功能清单

| 子功能 | 说明 |
|--------|------|
| 列表 | `GET /api/v1/alerts/channels`，分页/筛选以前端为准。 |
| 新建/编辑 | `POST` / `PUT /api/v1/alerts/channels/:id`，配置 URL、密钥、超时、启用状态等。 |
| 删除 | `DELETE /api/v1/alerts/channels/:id`。 |
| 连通测试 | `POST /api/v1/alerts/channels/:id/test`，发送探测消息。 |
| 字典辅助 | 部分字段（如钉钉加签）可从 **数据字典** 下拉选择，类型见运维在字典中维护的 `dict_type`。 |

## 3. 数据表

- 主表：`alert_channels`（及关联展示字段）。

## 4. 权限

- 需登录 + Casbin 中 `alerts/channels` 对应 `GET/POST/PUT/DELETE`。  
- 种子见 `cmd/seed.go` → `defaultPermissions`。

## 5. 注意事项

- **密钥不落日志**：前端输入框密码类字段避免在操作日志中明文打印（依赖后端脱敏策略）。  
- 与 **告警策略** 通过策略中的通道 ID 列表关联；删除通道前需确认无策略引用。  
- 邮件类 SMTP 全局参数可走 **数据字典优先、YAML 兜底**（见 `config.yaml` 注释）。
