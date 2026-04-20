# 菜单需求：菜单管理（`/menus`）

## 1. 定位

- **路由**：`/menus`，`MenusPage`。  
- **目标**：维护**动态侧边栏菜单树**（名称、路径、图标、`component` 懒加载名、排序、显隐）。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 树展示 | `GET /api/v1/menus/tree`（登录后拉取，无菜单权限时前端可用兜底路由）。 |
| 新建 | `POST /api/v1/menus`。 |
| 批量状态 | `PUT /api/v1/menus/status`。 |
| 更新 | `PUT /api/v1/menus/:id`。 |
| 删除 | `DELETE /api/v1/menus/:id`。 |

## 3. 数据表

- `menus`。

## 4. 注意事项

- **`component`** 必须与 `web/src/pages/**/*-page.tsx` 的 loader 命名一致（如 `projects-page`）。  
- 父级目录可仅作分组，`component` 可空。  
- 修改菜单后用户需刷新或重新登录以拉新树。
