# 菜单需求：项目成员（`/project-members`）

## 1. 定位

- **路由**：`/project-members`，`project-members-page`。  
- **目标**：在选定项目下维护 **project_members**（与全局 RBAC 独立），角色：`owner` / `admin` / `member` / `readonly`。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 选项目 | 前端拉取 `GET /api/v1/projects`。 |
| 成员列表 | `GET /api/v1/projects/:id/members`。 |
| 添加 | `POST /api/v1/projects/:id/members`。 |
| 改角色 | `PUT /api/v1/projects/:id/members/:memberId`。 |
| 移除 | `DELETE .../members/:memberId`。 |

## 3. 联动

- 监控规则若 `project_id` 匹配，则**启用成员邮箱**并入告警通知（与处理人去重）。详见 `AlertRuleAssigneeService.ResolveNotifyEmails`。

## 4. 注意事项

- 同一 `(project_id, user_id)` 唯一；重复添加会报错。  
- 项目列表页「成员」抽屉与本页共用 `ProjectMembersPanel` 逻辑。
