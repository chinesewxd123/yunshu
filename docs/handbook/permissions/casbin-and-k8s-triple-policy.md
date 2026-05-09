# 权限设计：Casbin API 与 K8s 三元策略

## 1. 模型

- 配置文件：`configs/casbin_model.conf`（随仓库提供）。  
- 适配器：GORM 表 `casbin_rule`（见 Casbin GORM Adapter）。

## 2. API 级权限（主体：用户 ID）

中间件：`internal/middleware/casbin.go` → `Authorize`

- **Subject**：`UserSubject(user.ID)`，与策略中角色/用户绑定方式以实际 `SyncUserRoles` 为准（常见为用户 ID 或角色分组，以项目 `policy_service` 同步逻辑为准）。
- **Object**：`c.FullPath()`，例如 `/api/v1/projects/:id/members`。
- **Action**：HTTP Method，如 `GET`、`POST`。

### 2.1 super-admin

角色码包含 `super-admin` 时 **不做 API 级 Casbin 校验**，直接放行（便于新接口上线前未录入策略时仍可管理）。

例外：当请求命中 **K8s 三元中间件**（路径已纳入 scope 目录且能解析出 `cluster_id`）时，**命名空间黑名单**先于 super-admin 的三元旁路执行；避免平台账号误操作受保护命名空间。

### 2.2 权限元数据

表 `permissions` 字段：

- `resource`：与 Gin `FullPath` 一致的路径模板  
- `action`：`GET` / `POST` / `PUT` / `DELETE` 等  

种子：`cmd/seed.go` 中 `defaultPermissions()`，运行 `go run . seed` 会 upsert 并给 `super-admin` 加策略。

## 3. K8s「三元」策略（集群 / 命名空间 / 资源路径）

用于**细粒度**授权：在「K8s 三元策略」页面为角色下发策略后，Casbin 中对象形如：

```text
k8s:cluster:{clusterID}:ns:{namespace}:{apiPath}
```

（具体拼接以 `internal/service/k8s_scoped_policy_service.go` 与 `internal/middleware/k8s_scope_authorize.go` 为准。）

### 3.1 与 k8m 多集群权限模型的对照（概念）

[k8m](https://github.com/weibaohui/k8m)（参见 [多集群 RBAC 与命名空间权限说明](https://zread.ai/weibaohui/k8m/20-multi-cluster-rbac-and-namespace-permissions)）采用 **平台管理员 / 集群角色（cluster_admin、cluster_readonly、cluster_pod_exec）+ 用户或用户组绑定 + 命名空间白名单/黑名单** 的一体化模型，并在权限检查中 **黑名单优先于白名单**。

yunshu 当前采用 **Casbin 两层**：

1. **API 权限**：`permissions` 表中的 `resource`（Gin FullPath）+ `action`（HTTP 方法或业务动作码）。
2. **K8s 三元策略**：对标记了 `k8s_scope_enabled` 的接口，再校验 `k8s:cluster:{id}:ns:{ns}:{path}` 与动作码；未配置任何三元策略时默认放行（见下）。

与 k8m 的差异与对应关系：

| k8m 能力 | yunshu 融合方案 |
|----------|----------------|
| 集群级粗粒度角色（只读 / 管理员 / Exec） | **预设档位** `POST /api/v1/k8s-policies/grant-preset`：`readonly`（资源 GET）、`readonly_exec`（只读 + exec 相关三元）、`admin`（纳入三元的高危变更 + 读路径）；由 `permissions` 表动态展开路径与动作码，避免手写上百条策略。 |
| 用户组绑定集群权限 | 不变：通过 **角色** 挂载三元策略；组授权映射到角色即可（与 k8m「组名一行权限」等价）。 |
| 命名空间 **黑名单** 优先 | `K8sScopeAuthorize` 内 **先于** `super-admin` 三元旁路执行；即黑名单对平台管理员在「已纳入三元目录的集群请求」上同样生效，防止误入生产命名空间。 |
| 命名空间 **白名单** 单独字段 | 仍为三元粒度：对选定集群/命名空间下发策略；不使用 `ns:*` 即等价收窄可见命名空间。 |

预设下发可选 **`deny_namespaces`**：在已选择**具体集群 ID**（非「全部集群」）时，同步写入 `k8s_namespace_deny_rules`，与页面手工添加黑名单一致。

### 3.2 读接口兜底

`allowReadByK8sScopedPolicy`：当用户 **无** 对应 API 的 Casbin 允许，但对 **GET** 请求且路径属于 K8s 资源读路径（与 `internal/service/k8s_read_paths.go` 中 `IsK8sReadAPIPath` 一致）时，若用户已有任意匹配的 `k8s:cluster:*` 权限，则允许读取（避免「已下发三元但仍 403」）。

例外：`/api/v1/menus/tree` 对 GET 永远放行（登录后需拉菜单）。

### 3.3 与 API 权限关系

- **写操作**（POST/PUT/DELETE）通常仍需要显式 API 权限或三元策略中的写动作（以实际模型为准）。  
- **集群列表** `GET /api/v1/clusters`：只要存在任意集群三元策略即可列表（代码注释已说明）。

### 3.4 与 k8m「资源 Watch」的对照（无 AI/MCP）

[k8m 文档](https://zread.ai/weibaohui/k8m/13-kubernetes-resource-watchers-and-event-handling) 描述的是**常驻**多资源 ListWatch、聚合缓存；代码里多用 **kom + 具体类型**（例如 `Resource(&v1.Pod{}).Watch(...)`），并非「HTTP 传入任意 GVR + RESTMapper 校验」这一通用形态。yunshu 未引入同等规模的常驻 Informer，改为 **按需** `GET /api/v1/k8s/resource-watch/stream`：**短名模式**仅需 `resource=pods` 等内置别名；**任意 GVR 模式**在同一路径上传 `group`（核心组可空）、`version`、`resource`（API 上的**复数名**），由当前集群 Discovery + `RESTMapper.ResourceFor` / `RESTMappings` 解析作用域并校验，再走 client-go dynamic `Watch` → SSE；鉴权仍为 `K8sScopeAuthorize` + 命名空间黑名单。扩展点见 `internal/pkg/eventbus`、`internal/pkg/extension`，**不是** k8m 插件状态机（参见 [插件](https://zread.ai/weibaohui/k8m/28-developing-custom-plugins) / [架构](https://zread.ai/weibaohui/k8m/6-system-architecture-and-design-philosophy)）。

## 4. 运维建议

- 新增 REST 路由后：更新 `defaultPermissions` + 运行 seed + 在「授权管理」给业务角色勾选。  
- 最小权限原则：生产环境避免给普通用户 `super-admin`。
