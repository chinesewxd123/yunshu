# 菜单需求：项目列表（`/projects`）

## 1. 定位

- **路由**：`/projects`，`projects-page`。  
- **目标**：**项目**租户 CRUD；创建者在后端自动写入 `project_members` 角色 **owner**。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 列表 | `GET /api/v1/projects`（分页、关键词）。 |
| 新建 | `POST /api/v1/projects`。 |
| 编辑 | `PUT /api/v1/projects/:id`。 |
| 删除 | `DELETE /api/v1/projects/:id`（会清理项目成员等关联，见服务实现）。 |
| 成员抽屉 | `GET/POST/PUT/DELETE /api/v1/projects/:id/members...`（详见项目成员文档）。 |

## 3. 数据表

- `projects`、`project_members`。

## 4. 注意事项

- **code** 一般唯一，用于租户标识。  
- 删除项目为高危操作，需权限与二次确认。
