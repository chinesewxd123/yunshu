# 菜单级需求文档索引

以下文档与 **数据库种子菜单**（`cmd/seed.go` → `defaultMenus`）、**前端路由**（`web/src/app/app.tsx`、`dynamic-menu-page` 兜底、`admin-layout` 固定项）对齐。  
OpenAPI 全量路由见 **`docs/apipost/permission-system.openapi.yaml`**（由 `go run ./tools/genopenapi` 从 `internal/router/router.go` 生成）。

## 一级入口与固定路由

| 路由 | 文档 |
|------|------|
| `/` 资产总览 | [menu-root-dashboard.md](./menu-root-dashboard.md) |
| `/login` 登录页 | [menu-login.md](./menu-login.md) |
| `/users` 账号管理 | [menu-users.md](./menu-users.md) |
| `/departments` 组织架构 | [menu-departments.md](./menu-departments.md) |
| `/roles` | [menu-roles.md](./menu-roles.md) |
| `/permissions` | [menu-permissions.md](./menu-permissions.md) |
| `/policies` | [menu-policies.md](./menu-policies.md) |
| `/registrations` | [menu-registrations.md](./menu-registrations.md) |
| `/menus` | [menu-menus.md](./menu-menus.md) |
| `/login-logs` | [menu-login-logs.md](./menu-login-logs.md) |
| `/operation-logs` | [menu-operation-logs.md](./menu-operation-logs.md) |
| `/banned-ips` | [menu-banned-ips.md](./menu-banned-ips.md) |
| `/clusters` | [menu-clusters.md](./menu-clusters.md) |
| `/pods` | [menu-pods.md](./menu-pods.md) |
| `/server-console` | [menu-server-console.md](./menu-server-console.md) |
| `/personal-settings` | [menu-personal-settings.md](./menu-personal-settings.md) |
| `/dict-entries`（顶栏） | [menu-dict-entries.md](./menu-dict-entries.md) |

## 告警通知

| 路由 | 文档 |
|------|------|
| `/alert-channels` | [menu-alert-channels.md](./menu-alert-channels.md) |
| `/alert-monitor-platform` | [menu-alert-monitor-platform.md](./menu-alert-monitor-platform.md) |
| `/alert-config-center` | [menu-alert-config-center.md](./menu-alert-config-center.md) |
| `/alert-events` | [menu-alert-events.md](./menu-alert-events.md) |
| `/alert-duty` | [menu-alert-duty.md](./menu-alert-duty.md) |

## 项目管理

| 路由 | 文档 |
|------|------|
| `/projects` | [menu-projects.md](./menu-projects.md) |
| `/project-members` | [menu-project-members.md](./menu-project-members.md) |
| `/project-servers` | [menu-project-servers.md](./menu-project-servers.md) |
| `/project-services` | [menu-project-services.md](./menu-project-services.md) |
| `/project-log-sources` | [menu-project-log-sources.md](./menu-project-log-sources.md) |
| `/project-logs` | [menu-project-logs.md](./menu-project-logs.md) |
| `/agent-list` | [menu-agent-list.md](./menu-agent-list.md) |

## 系统管理（种子菜单）

与上表 `/users`…`/banned-ips` 一致；另含：

| 路由 | 文档 |
|------|------|
| `/k8s-scoped-policies` | [menu-k8s-scoped-policies.md](./menu-k8s-scoped-policies.md) |

## Kubernetes 容器管理

列表/详情/YAML 等共性交互与权限模式见 **[menu-k8s-resource-pattern.md](./menu-k8s-resource-pattern.md)**；下表为各路由专用说明。

| 路由 | 文档 |
|------|------|
| `/namespaces` | [menu-namespaces.md](./menu-namespaces.md) |
| `/nodes` | [menu-nodes.md](./menu-nodes.md) |
| `/component-status` | [menu-component-status.md](./menu-component-status.md) |
| `/deployments` | [menu-deployments.md](./menu-deployments.md) |
| `/statefulsets` | [menu-statefulsets.md](./menu-statefulsets.md) |
| `/daemonsets` | [menu-daemonsets.md](./menu-daemonsets.md) |
| `/cronjobs` | [menu-cronjobs.md](./menu-cronjobs.md) |
| `/jobs` | [menu-jobs.md](./menu-jobs.md) |
| `/configmaps` | [menu-configmaps.md](./menu-configmaps.md) |
| `/secrets` | [menu-secrets.md](./menu-secrets.md) |
| `/k8s-services` | [menu-k8s-services.md](./menu-k8s-services.md) |
| `/persistentvolumes` | [menu-persistentvolumes.md](./menu-persistentvolumes.md) |
| `/persistentvolumeclaims` | [menu-persistentvolumeclaims.md](./menu-persistentvolumeclaims.md) |
| `/storageclasses` | [menu-storageclasses.md](./menu-storageclasses.md) |
| `/ingresses` | [menu-ingresses.md](./menu-ingresses.md) |
| `/ingress-classes` | [menu-ingress-classes.md](./menu-ingress-classes.md) |
| `/events` | [menu-events.md](./menu-events.md) |
| `/rbac/roles` | [menu-rbac-roles.md](./menu-rbac-roles.md) |
| `/rbac/rolebindings` | [menu-rbac-rolebindings.md](./menu-rbac-rolebindings.md) |
| `/rbac/clusterroles` | [menu-rbac-clusterroles.md](./menu-rbac-clusterroles.md) |
| `/rbac/clusterrolebindings` | [menu-rbac-clusterrolebindings.md](./menu-rbac-clusterrolebindings.md) |

## Kubernetes CRD 管理

| 路由 | 文档 |
|------|------|
| `/crds` | [menu-crds.md](./menu-crds.md) |
| `/crs` | [menu-crs.md](./menu-crs.md) |

## 其它

| 路由 | 文档 |
|------|------|
| `/runtime-config` | [menu-runtime-config.md](./menu-runtime-config.md) |
