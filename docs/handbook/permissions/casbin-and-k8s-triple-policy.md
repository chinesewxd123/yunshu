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

角色码包含 `super-admin` 时 **不做 Casbin 校验**，直接放行（便于新接口上线前未录入策略时仍可管理）。

### 2.2 权限元数据

表 `permissions` 字段：

- `resource`：与 Gin `FullPath` 一致的路径模板  
- `action`：`GET` / `POST` / `PUT` / `DELETE` 等  

种子：`cmd/seed.go` 中 `defaultPermissions()`，运行 `go run . seed` 会 upsert 并给 `super-admin` 加策略。

## 3. K8s「三元」策略（集群 / 命名空间 / 资源路径）

用于**细粒度**授权：在「K8s 三元策略」页面为角色下发策略后，Casbin 中对象形如：

```text
k8s:cluster:{clusterID}:r:{namespace}:{apiPath}
```

（具体拼接以 `internal/service/k8s_scoped_policy_service.go` 为准。）

### 3.1 读接口兜底

`allowReadByK8sScopedPolicy`：当用户 **无** 对应 API 的 Casbin 允许，但对 **GET** 请求且路径属于 K8s 资源读路径时，若用户已有任意匹配的 `k8s:cluster:*` 权限，则允许读取（避免「已下发三元但仍 403」）。

例外：`/api/v1/menus/tree` 对 GET 永远放行（登录后需拉菜单）。

### 3.2 与 API 权限关系

- **写操作**（POST/PUT/DELETE）通常仍需要显式 API 权限或三元策略中的写动作（以实际模型为准）。  
- **集群列表** `GET /api/v1/clusters`：只要存在任意集群三元策略即可列表（代码注释已说明）。

## 4. 运维建议

- 新增 REST 路由后：更新 `defaultPermissions` + 运行 seed + 在「授权管理」给业务角色勾选。  
- 最小权限原则：生产环境避免给普通用户 `super-admin`。
