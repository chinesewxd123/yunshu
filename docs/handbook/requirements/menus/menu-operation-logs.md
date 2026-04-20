# 菜单需求：操作历史（`/operation-logs`）

## 1. 定位

- **路由**：`/operation-logs`，`OperationLogsPage`。  
- **目标**：记录用户在平台内**写操作**（依赖 `OperationAudit` 中间件的路由）。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 分页列表 | `GET /api/v1/operation-logs`。 |
| 导出 | `GET /api/v1/operation-logs/export`。 |
| 删除 | `DELETE /api/v1/operation-logs/:id`、`POST .../delete` 批量。 |

## 3. 数据表

- `operation_logs`。

## 4. 注意事项

- 并非所有只读接口都会产生记录；以中间件挂载范围为准。  
- 敏感字段应在落库前脱敏。
