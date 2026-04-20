# 菜单需求：登录日志（`/login-logs`）

## 1. 定位

- **路由**：`/login-logs`，`LoginLogsPage`。  
- **目标**：审计**登录行为**（成功/失败、IP、UA、时间等）。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 分页列表 | `GET /api/v1/login-logs`。 |
| 导出 | `GET /api/v1/login-logs/export`。 |
| 单条删除 | `DELETE /api/v1/login-logs/:id`。 |
| 批量删除 | `POST /api/v1/login-logs/delete`。 |

## 3. 数据表

- `login_logs`。

## 4. 注意事项

- 量大时建议按时间筛选与定期归档。  
- 与 **封禁 IP** 策略可联动分析暴力破解。
