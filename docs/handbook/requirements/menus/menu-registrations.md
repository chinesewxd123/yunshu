# 菜单需求：注册审核（`/registrations`）

## 1. 定位

- **路由**：`/registrations`，`RegistrationsPage`。  
- **目标**：处理 `POST /api/v1/auth/register` 产生的**待审核申请**，通过后创建用户。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 待审列表 | `GET /api/v1/registrations`。 |
| 审核 | `POST /api/v1/registrations/:id/review`，通过/拒绝并可选备注。 |

## 3. 数据表

- `registration_requests`。

## 4. 注意事项

- 通过后用户默认角色与初始密码策略由后端实现决定。  
- 资产总览「待审核」数量依赖该表状态统计。
