# 权限设计：Casbin API 与 K8s 三元策略

## 1. 模型

- 配置文件：`configs/casbin_model.conf`（随仓库提供）。  
- 适配器：GORM 表 `casbin_rule`（见 Casbin GORM Adapter）。

## 2. API 级权限（主体：用户 ID）

中间件：`internal/middleware/casbin.go` → `Authorize`

- **Subject**：`UserSubject(user.ID)`，形如 `user:{id}`（见 `internal/service/casbin_sync.go`）。
- **Object**：`c.FullPath()`，例如 `/api/v1/projects/:id/members`。
- **Action**：HTTP Method，如 `GET`、`POST`。

用户与角色码的绑定：`SyncUserRoles` 为每个用户建立 `g(user:123, viewer)` 这类分组；**API 级 `p` 策略**通常挂在 **角色码**（如 `viewer`）或资源上，由「授权管理」界面维护。

### 2.1 super-admin

角色码包含 `super-admin` 时 **不做 API 级 Casbin 校验**，直接放行（便于新接口上线前未录入策略时仍可管理）。

例外：当请求命中 **K8s 三元中间件**（路径已纳入 scope 目录且能解析出 `cluster_id`）时，**命名空间黑名单**先于 super-admin 的三元旁路执行；避免平台账号误操作受保护命名空间。

### 2.2 权限元数据

表 `permissions` 字段：

- `resource`：与 Gin `FullPath` 一致的路径模板  
- `action`：`GET` / `POST` / `PUT` / `DELETE` 等  

种子：`cmd/seed.go` 中 `defaultPermissions()`，运行 `go run . seed` 会 upsert 并给 `super-admin` 加策略。

### 2.3 K8s 三元策略挂在谁身上？

- **三元策略**在 Casbin 里以 **`(角色码, k8s:cluster:…:ns:…:/api/…, 动作码)`** 的 `p` 策略形式存在（第一列为角色码，与「K8s 三元策略」页面所选角色一致）。
- 鉴权中间件每次从数据库加载用户及其角色，填充 `CurrentUser.RoleCodes`（见 `internal/middleware/auth.go`）；中间件里用 `UserSubject(user.ID)` 做 `Enforce`，Casbin 通过 **用户→角色** 分组继承上述 `p` 策略。
- **结论**：在「用户管理 / 角色管理」里给用户绑定 **业务角色**（如 `viewer`）后，该用户才继承该角色下的三元策略与（见下）命名空间黑名单规则。

## 3. K8s「三元」策略（集群 / 命名空间 / 资源路径）

用于**细粒度**授权：在「K8s 三元策略」页面为角色下发策略后，Casbin 中对象形如：

```text
k8s:cluster:{clusterID}:ns:{namespace}:{apiPath}
```

（具体拼接以 `internal/service/k8s_scoped_policy_service.go` 与 `internal/middleware/k8s_scope_authorize.go` 为准。）

`clusterID` 为 `0` 或 `*` 时表示「不限制集群」维度，由中间件按多种资源串组合尝试匹配（见 `buildK8sScopeResource` 调用处）。

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

### 3.5 K8s 三元中间件：判定顺序与「默认放行」

实现见 `internal/middleware/k8s_scope_authorize.go`，对运维理解行为很重要：

1. **未纳入三元目录**的路由（`permissions` 未标记 `k8s_scope_enabled` 或未出现在映射里）→ **不拦截**，直接 `Next`。
2. 从请求中解析 **`cluster_id`** 与 **`namespace`**：支持 Query、`id` 路由参数、JSON Body（`cluster_id` / `clusterId`、`namespace`）等；若 **`cluster_id == 0`** → **不拦截**（兼容旧接口）。
3. **命名空间黑名单**（表 `k8s_namespace_deny_rules`）：当 `namespace` 非空且不为 `_cluster` 时，按 **鉴权上下文中用户的任一角色码 + 集群 ID + 命名空间** 查询；命中则 **403**（**含 super-admin**）。
4. **`super-admin`**：通过黑名单后 **跳过后续三元 Enforce**，直接放行。
5. **其他用户**：在多条候选资源串上执行 `Enforce`；若 Casbin 中存在与该资源串+动作相关的 **`p` 策略**（`hasScopedPolicy`）但当前用户不满足 → **403**；若 **不存在** 相关策略 → **默认放行**（注释中的「兼容旧权限」行为）。

**实践含义**：生产环境一旦开始为某类 K8s 接口广泛配置三元策略，应同步保证业务角色均有明确、最小化的三元条目，避免依赖「无策略则放行」的宽松路径。

## 4. 配置前准备（检查清单）

| 步骤 | 说明 |
|------|------|
| 确认平台集群 ID | 黑名单、按集群下发三元时，需使用 **平台「集群管理」中的数字 ID**；黑名单唯一键为 `(role_code, cluster_id, namespace)`。 |
| 选定角色码 | 三元策略与黑名单均按 **角色编码**（如 `viewer`）配置；勿与显示名称混淆。 |
| 绑定用户 | 在 **用户管理 → 分配角色** 或 **角色管理 → 分配用户** 中，将目标用户绑定到上一步角色。服务端鉴权从数据库加载用户与角色（见 `internal/middleware/auth.go`），**角色变更会在该用户下一次带 Token 的请求起生效**；若前端缓存了「当前用户」资料，可刷新页面或重新拉取 `/auth/me`。 |
| API 级菜单/授权 | 控制台菜单仍受 API 级 Casbin 控制；仅配置三元而未在「授权管理」勾选对应 API 时，用户可能仍无法进入某页（与 `allowReadByK8sScopedPolicy` 等兜底规则并存，见 3.2）。 |

## 5. 场景化配置指导

以下场景均在 **「K8s 三元策略」** 相关页面或等价 API 完成；命名空间黑名单亦可在同页底部 **「命名空间黑名单」** 区块维护。

### 场景 A：禁止查看与操作若干命名空间（黑名单，推荐用于「生产 / kube-system 隔离」）

**目标**：某业务角色在 **指定集群** 下完全不能访问 `kube-system`、`production` 等命名空间（列表、详情、Exec 等凡带该 `namespace` 的请求均拒绝）。

**配置步骤**：

1. 打开 **命名空间黑名单**，选择角色码（如 `viewer`）。
2. **集群** 选择 **具体集群**（必须为平台登记 ID；全集群通配不会写入本表的有效语义）。
3. 命名空间填写 `kube-system` → 添加；对 `production` 重复添加多条规则。
4. 将需要受限的用户绑定到该角色。

**验证**：以该用户登录，在控制台选中上述集群并切换到被禁命名空间，应收到类似「当前角色在此集群下禁止访问命名空间「xxx」」的 403。

**说明**：黑名单 **优先于** 三元白名单；且对 **super-admin** 同样生效（见 2.1），用于防止平台账号误入关键命名空间。

---

### 场景 B：仅允许少数命名空间（白名单式收窄，不用「全 NS + 黑名单」）

**目标**：用户只能访问 `app-ns-a`、`app-ns-b`，其它命名空间一律不可见或不可操作。

**配置步骤**：

1. 审视该角色在三元表格中是否存在大量 **`命名空间 = *`** 的策略；若有，表示对所有 NS 放开该路径，与「白名单」目标相反。
2. **删除** 或不再使用「全部命名空间」的宽策略（按行删除或重建角色策略集）。
3. 使用 **「快速下发」**：**集群** 选目标集群，**命名空间** 依次填 `app-ns-a`，勾选所需 **资源路径**（如 `/api/v1/pods`、`/api/v1/deployments`），**动作** 仅勾选业务需要项（通常只读为 `GET`）；对 `app-ns-b` 重复。
4. 不对其它命名空间下发任何三元策略。

**验证**：切换未授权的命名空间时，应无读权限或在三元强制路径上 403。

**说明**：本项目的三元键包含 **命名空间维度**，收窄 NS 即收窄可见资源范围；与 k8m「白名单」概念对齐，但实现为多条 Casbin `p` 策略而非单独字段。

---

### 场景 C：能看工作负载与 Pod，但不能 Exec、不能删 Pod

**目标**：保留控制台只读浏览，禁止终端 / Exec、删除 Pod 等高危操作。

**配置步骤**：

1. 使用预设 **`readonly`** 下发（`POST /api/v1/k8s-policies/grant-preset` 或页面「只读」档位），**不要**选择 `readonly_exec` 或 `admin`（档位语义见 `internal/service/k8s_cluster_preset.go`）。
2. 若历史上误发了 Exec 相关策略，在三元策略表中 **删除** 动作含 `exec` 或与 Exec 接口相关的行。
3. 在 **授权管理** 中避免为该角色勾选 Pod 删除、Exec 等写类 API（与 3.3 一致）。

**验证**：只读列表与详情可用；打开终端或删除 Pod 应失败。

---

### 场景 D：只读浏览 + 仅对部分命名空间开放 Exec

**目标**：在 `dev` 命名空间允许 Exec，在 `staging` 只读，生产命名空间禁止。

**配置思路**（组合使用）：

1. 对 **`dev`** 命名空间：使用 **`readonly_exec`** 档位或手动为 Exec 相关路径追加动作（以权限目录中实际路径为准）。
2. 对 **`staging`**：仅 `readonly` 或手动只勾选 `GET`。
3. 对 **`production`**：使用 **场景 A 黑名单** 直接禁止整个 NS，避免误配 Exec。

**说明**：档位是按「集群 × 命名空间列表」批量展开的；多环境组合时通常 **分多次下发** 或拆分多个角色更易维护。

---

### 场景 E：档位一键下发并同步写入多条黑名单

**目标**：给 `viewer` 批量下发只读三元，同时在集群 `5` 上禁止 `kube-system`、`monitoring`。

**配置步骤**（与 `K8sScopedPolicyGrantPresetRequest` 字段一致）：

1. 调用 **`POST /api/v1/k8s-policies/grant-preset`** 或在页面使用 **按档位下发**。
2. `preset` 选 `readonly`；**`cluster_ids` 必须为具体 ID 列表**（不可依赖「全部集群」占位来写黑名单）。
3. `deny_namespaces` 填 `["kube-system","monitoring"]`（或页面等价多选/逐条输入）。

**说明**：`deny_namespaces` 仅在选中 **明确集群 ID** 时才会写入 `k8s_namespace_deny_rules`；与在黑名单 UI 手工添加等价。

---

### 场景 F：多集群只读 + 每集群不同黑名单

**目标**：同一 `viewer` 在集群 A 禁止 `default`，在集群 B 禁止 `kube-system`。

**配置步骤**：在黑名单中 **按集群分别添加** 规则（每条规则绑定一个 `cluster_id`）；同一角色码可有多行。

---

### 场景 G：验证「读接口兜底」与集群列表

**目标**：理解为何「只有部分三元」时仍可能拉通部分 GET。

**要点**（见 3.2）：对符合 `IsK8sReadAPIPath` 的 **GET**，若用户已有任意 `k8s:cluster:*` 相关隐式权限，可能通过 `allowReadByK8sScopedPolicy` 获得读取能力；**集群列表** `GET /api/v1/clusters` 在「存在任意集群三元策略」时亦可列出。

**运维建议**：若希望 **严格禁止** 某读接口，应在 API 级显式拒绝或收紧三元与菜单权限，而不要仅依赖「未勾选菜单」的单一手段。

## 6. 能力边界与常见误区

| 话题 | 说明 |
|------|------|
| **能否按 Pod 名称单独授权？** | **不能。** 鉴权维度为 **集群 + 命名空间 + 平台 API 路径 + 动作码**；请求体中的 Pod 名不参与三元串。若需「同一 NS 内不同 Pod 不同权限」，应在集群侧使用 K8s RBAC、Admission、OPA 等方案。 |
| **黑名单是否区分 HTTP 动作？** | 否。命中黑名单后该命名空间下相关请求一律拒绝。 |
| **`namespace` 为空时** | 中间件会规范为 `_cluster`；黑名单 **不** 对 `_cluster` 生效（见 `IsDenied` 实现）。 |
| **只配黑名单不配三元** | 黑名单仍可拦截；但若用户无任何 API 与菜单权限，可能仍无法进入控制台，需结合「授权管理」配置。 |
| **依赖「无三元策略则放行」** | 见 3.5：在已存在全局三元 `p` 策略的前提下，未给某用户组配置对应继承关系可能导致意外放行或拒绝，应通过 **明确角色模板** 管理。 |

## 7. 运维建议

- 新增 REST 路由后：更新 `defaultPermissions` + 运行 seed + 在「授权管理」给业务角色勾选。  
- 最小权限原则：生产环境避免给普通用户 `super-admin`；K8s 生产命名空间优先用 **黑名单** 或 **窄命名空间三元** 双重保险。
- 变更三元策略或 Casbin 规则后，若用户仍报权限异常，可让其 **刷新页面** 或 **重新登录** 以排除前端状态缓存；服务端中间件每次请求会重载用户角色（见 `auth` 中间件）。
