# 菜单需求：数据字典（`/dict-entries`）

## 1. 定位

- **路由**：`/dict-entries`，顶栏固定入口，`dict-entries-page`。  
- **目标**：按 **dict_type** 维护键值型配置，供告警通道 URL、邮件参数、Prometheus 地址、K8s 模板等多处「字典优先、YAML 兜底」使用。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/dict/entries`。 |
| 新建/更新/删除 | `POST`、`PUT /:id`、`DELETE /:id`。 |
| 下拉选项 | `GET /api/v1/dict/options/:dictType`。 |

## 3. 数据表

- `dict_entries`；启动迁移含去重逻辑（见 `migrate_schema.go`）。

## 4. 注意事项

- 修改**已启用**字典项会影响线上行为（如 SMTP、Webhook）。  
- `dict_type` 命名需与代码/文档约定一致（如 `mail_host`、`alert_datasource_base_url`）。
