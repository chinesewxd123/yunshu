/**
 * 与后端 `internal/router/router.go` 中 `/api/v1` 路由对齐，便于在控制台核对「接口 ↔ 页面」映射。
 * 若后端增删路由，请同步更新本文件与 seed 中的 Casbin 能力项。
 */
export type ApiCatalogRow = {
  method: string;
  path: string;
  summary: string;
  /** 对应前端入口；无独立页面时说明调用场景 */
  ui: string;
  /** 是否需要 Bearer Token（健康检查等除外） */
  auth: boolean;
};

export const API_CATALOG_GROUPS: { title: string; routes: ApiCatalogRow[] }[] = [
  {
    title: "系统",
    routes: [
      {
        method: "GET",
        path: "/api/v1/health",
        summary: "健康检查与进程运行信息",
        ui: "资产总览 · 系统状态",
        auth: false,
      },
    ],
  },
  {
    title: "认证（多数无需登录）",
    routes: [
      { method: "POST", path: "/api/v1/auth/verification-code", summary: "发送邮箱验证码", ui: "登录页 / 注册页", auth: false },
      {
        method: "POST",
        path: "/api/v1/auth/login-code",
        summary: "按用户名向绑定邮箱发登录验证码",
        ui: "可与邮箱登录配合（前端当前走邮箱直发）",
        auth: false,
      },
      { method: "POST", path: "/api/v1/auth/password-login-code", summary: "获取图形验证码", ui: "登录页 · 用户名密码", auth: false },
      { method: "POST", path: "/api/v1/auth/login", summary: "用户名密码登录", ui: "登录页 · 用户名密码", auth: false },
      { method: "POST", path: "/api/v1/auth/email-login", summary: "邮箱验证码登录", ui: "登录页 · 邮箱验证码", auth: false },
      { method: "POST", path: "/api/v1/auth/register", summary: "提交注册申请", ui: "登录页 · 注册账号", auth: false },
      { method: "POST", path: "/api/v1/auth/logout", summary: "注销当前 Token", ui: "顶栏用户菜单 · 退出登录", auth: true },
      { method: "GET", path: "/api/v1/auth/me", summary: "当前用户与角色", ui: "全局（进入控制台时拉取）", auth: true },
    ],
  },
  {
    title: "账号",
    routes: [
      { method: "GET", path: "/api/v1/users", summary: "分页列表", ui: "账号管理", auth: true },
      { method: "POST", path: "/api/v1/users", summary: "创建用户", ui: "账号管理", auth: true },
      { method: "GET", path: "/api/v1/users/:id", summary: "用户详情", ui: "账号管理", auth: true },
      { method: "PUT", path: "/api/v1/users/:id", summary: "更新用户", ui: "账号管理", auth: true },
      { method: "DELETE", path: "/api/v1/users/:id", summary: "删除用户", ui: "账号管理", auth: true },
      { method: "PUT", path: "/api/v1/users/:id/roles", summary: "分配角色", ui: "账号管理", auth: true },
    ],
  },
  {
    title: "角色",
    routes: [
      { method: "GET", path: "/api/v1/roles", summary: "分页列表", ui: "角色管理", auth: true },
      { method: "POST", path: "/api/v1/roles", summary: "创建角色", ui: "角色管理", auth: true },
      { method: "GET", path: "/api/v1/roles/:id", summary: "角色详情", ui: "角色管理", auth: true },
      { method: "PUT", path: "/api/v1/roles/:id", summary: "更新角色", ui: "角色管理", auth: true },
      { method: "DELETE", path: "/api/v1/roles/:id", summary: "删除角色", ui: "角色管理", auth: true },
    ],
  },
  {
    title: "接口能力（权限元数据）",
    routes: [
      { method: "GET", path: "/api/v1/permissions", summary: "分页列表", ui: "API 管理", auth: true },
      { method: "POST", path: "/api/v1/permissions", summary: "新建能力项", ui: "API 管理", auth: true },
      { method: "GET", path: "/api/v1/permissions/:id", summary: "详情", ui: "API 管理", auth: true },
      { method: "PUT", path: "/api/v1/permissions/:id", summary: "更新", ui: "API 管理", auth: true },
      { method: "DELETE", path: "/api/v1/permissions/:id", summary: "删除", ui: "API 管理", auth: true },
    ],
  },
  {
    title: "授权策略（Casbin）",
    routes: [
      { method: "GET", path: "/api/v1/policies", summary: "策略列表", ui: "授权管理", auth: true },
      { method: "POST", path: "/api/v1/policies", summary: "授予：角色绑定能力项", ui: "授权管理", auth: true },
      {
        method: "DELETE",
        path: "/api/v1/policies",
        summary: "撤销：请求体携带 role_id + permission_id",
        ui: "授权管理",
        auth: true,
      },
    ],
  },
  {
    title: "注册审核",
    routes: [
      { method: "GET", path: "/api/v1/registrations", summary: "申请列表", ui: "注册审核", auth: true },
      { method: "POST", path: "/api/v1/registrations/:id/review", summary: "审核", ui: "注册审核", auth: true },
    ],
  },
  {
    title: "菜单",
    routes: [
      { method: "GET", path: "/api/v1/menus/tree", summary: "菜单树", ui: "菜单管理", auth: true },
      { method: "POST", path: "/api/v1/menus", summary: "创建菜单", ui: "菜单管理", auth: true },
      { method: "PUT", path: "/api/v1/menus/:id", summary: "更新菜单", ui: "菜单管理", auth: true },
      { method: "DELETE", path: "/api/v1/menus/:id", summary: "删除菜单", ui: "菜单管理", auth: true },
    ],
  },
  {
    title: "审计",
    routes: [
      { method: "GET", path: "/api/v1/login-logs", summary: "登录日志", ui: "登录日志", auth: true },
      { method: "DELETE", path: "/api/v1/login-logs/:id", summary: "删除单条", ui: "登录日志", auth: true },
      { method: "POST", path: "/api/v1/login-logs/delete", summary: "批量删除", ui: "登录日志", auth: true },
      { method: "GET", path: "/api/v1/operation-logs", summary: "操作历史", ui: "操作历史", auth: true },
      { method: "DELETE", path: "/api/v1/operation-logs/:id", summary: "删除单条", ui: "操作历史", auth: true },
      { method: "POST", path: "/api/v1/operation-logs/delete", summary: "批量删除", ui: "操作历史", auth: true },
    ],
  },
];
