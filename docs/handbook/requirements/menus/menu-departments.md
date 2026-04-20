# 菜单需求：组织架构（`/departments`）

## 1. 定位

- **路由**：`/departments`，`DepartmentsPage`。  
- **目标**：维护**部门树**，供用户归属、告警处理人「按部门子树展开用户」等能力使用。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 树形展示 | `GET /api/v1/departments/tree`。 |
| 节点详情 | `GET /api/v1/departments/:id`。 |
| 新增子部门 | `POST /api/v1/departments`。 |
| 编辑 | `PUT /api/v1/departments/:id`。 |
| 删除 | `DELETE /api/v1/departments/:id`（需确认子部门与用户约束）。 |

## 3. 数据表

- `departments`；用户表 `department_id` 关联。

## 4. 注意事项

- **告警监控规则处理人**中选择部门时，后端按**子树**解析启用用户邮箱；部门调整会影响通知范围。  
- 删除部门前需迁移或解除用户、子部门依赖。
