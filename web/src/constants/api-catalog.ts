/**
 * 与后端 `internal/router/router.go` 中 `/api/v1` 路由对齐，便于在控制台核对「接口 ↔ 页面」映射。
 * 若后端增删路由，请同步更新本文件与 `cmd/seed.go` 中 Casbin 能力项（`Name` 与本文件 `summary` 保持一致，便于 API 管理页「一键补全接口」与权限名对齐）。
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
      { method: "PUT", path: "/api/v1/auth/me", summary: "更新个人资料", ui: "个人设置", auth: true },
      { method: "PUT", path: "/api/v1/auth/password", summary: "修改个人密码", ui: "个人设置", auth: true },
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
      { method: "GET", path: "/api/v1/users/import-template", summary: "下载用户导入模板", ui: "账号管理", auth: true },
    ],
  },
  {
    title: "组织架构",
    routes: [
      { method: "GET", path: "/api/v1/departments/tree", summary: "部门树", ui: "组织架构", auth: true },
      { method: "GET", path: "/api/v1/departments/:id", summary: "部门详情", ui: "组织架构", auth: true },
      { method: "POST", path: "/api/v1/departments", summary: "创建部门", ui: "组织架构", auth: true },
      { method: "PUT", path: "/api/v1/departments/:id", summary: "更新部门（支持迁移子树）", ui: "组织架构", auth: true },
      { method: "DELETE", path: "/api/v1/departments/:id", summary: "删除部门", ui: "组织架构", auth: true },
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
  {
    title: "告警中心",
    routes: [
      {
        method: "POST",
        path: "/api/v1/alerts/webhook/alertmanager",
        summary: "接收 Alertmanager Webhook",
        ui: "外部系统推送",
        auth: false,
      },
      { method: "GET", path: "/api/v1/alerts/channels", summary: "告警通道列表", ui: "告警通道", auth: true },
      { method: "POST", path: "/api/v1/alerts/channels", summary: "创建告警通道", ui: "告警通道", auth: true },
      { method: "PUT", path: "/api/v1/alerts/channels/:id", summary: "更新告警通道", ui: "告警通道", auth: true },
      { method: "DELETE", path: "/api/v1/alerts/channels/:id", summary: "删除告警通道", ui: "告警通道", auth: true },
      { method: "POST", path: "/api/v1/alerts/channels/:id/test", summary: "测试告警通道", ui: "告警通道", auth: true },
      { method: "GET", path: "/api/v1/alerts/events", summary: "告警事件列表", ui: "历史告警记录", auth: true },
      { method: "GET", path: "/api/v1/alerts/history/stats", summary: "告警历史统计", ui: "告警配置中心", auth: true },
      { method: "GET", path: "/api/v1/alerts/datasources", summary: "告警数据源列表", ui: "告警监控平台 · 数据源", auth: true },
      { method: "POST", path: "/api/v1/alerts/datasources", summary: "创建告警数据源", ui: "告警监控平台 · 数据源", auth: true },
      { method: "PUT", path: "/api/v1/alerts/datasources/:id", summary: "更新告警数据源", ui: "告警监控平台 · 数据源", auth: true },
      { method: "DELETE", path: "/api/v1/alerts/datasources/:id", summary: "删除告警数据源", ui: "告警监控平台 · 数据源", auth: true },
      { method: "GET", path: "/api/v1/alerts/datasources/:id/prometheus-alerts", summary: "Prometheus 活跃告警快照", ui: "告警监控平台 · 静默", auth: true },
      { method: "POST", path: "/api/v1/alerts/datasources/:id/query", summary: "PromQL 即时查询", ui: "告警监控平台 · PromQL 控制台", auth: true },
      { method: "POST", path: "/api/v1/alerts/datasources/:id/query_range", summary: "PromQL 范围查询", ui: "告警监控平台 · PromQL 控制台", auth: true },
      { method: "GET", path: "/api/v1/alerts/silences", summary: "告警静默列表", ui: "告警监控平台 · 静默", auth: true },
      { method: "POST", path: "/api/v1/alerts/silences", summary: "创建告警静默", ui: "告警监控平台 · 静默", auth: true },
      { method: "POST", path: "/api/v1/alerts/silences/batch", summary: "批量创建告警静默", ui: "告警监控平台 · 静默", auth: true },
      { method: "PUT", path: "/api/v1/alerts/silences/:id", summary: "更新告警静默", ui: "告警监控平台 · 静默", auth: true },
      { method: "DELETE", path: "/api/v1/alerts/silences/:id", summary: "删除告警静默", ui: "告警监控平台 · 静默", auth: true },
      { method: "GET", path: "/api/v1/alerts/monitor-rules", summary: "监控告警规则列表", ui: "告警监控平台 · 监控规则", auth: true },
      { method: "POST", path: "/api/v1/alerts/monitor-rules", summary: "创建监控告警规则", ui: "告警监控平台 · 监控规则", auth: true },
      { method: "PUT", path: "/api/v1/alerts/monitor-rules/:id", summary: "更新监控告警规则", ui: "告警监控平台 · 监控规则", auth: true },
      { method: "DELETE", path: "/api/v1/alerts/monitor-rules/:id", summary: "删除监控告警规则", ui: "告警监控平台 · 监控规则", auth: true },
      { method: "GET", path: "/api/v1/alerts/monitor-rules/:id/assignees", summary: "监控规则处理人", ui: "告警监控平台 · 处理人", auth: true },
      { method: "PUT", path: "/api/v1/alerts/monitor-rules/:id/assignees", summary: "配置监控规则处理人", ui: "告警监控平台 · 处理人", auth: true },
      { method: "GET", path: "/api/v1/alerts/duty-blocks", summary: "规则值班班次列表（按 monitor_rule_id）", ui: "告警监控平台 · 规则与值班绑定", auth: true },
      { method: "POST", path: "/api/v1/alerts/duty-blocks", summary: "创建规则值班班次", ui: "告警监控平台 · 规则与值班绑定", auth: true },
      { method: "PUT", path: "/api/v1/alerts/duty-blocks/:id", summary: "更新规则值班班次", ui: "告警监控平台 · 规则与值班绑定", auth: true },
      { method: "DELETE", path: "/api/v1/alerts/duty-blocks/:id", summary: "删除规则值班班次", ui: "告警监控平台 · 规则与值班绑定", auth: true },
      { method: "POST", path: "/api/v1/agents/health/report", summary: "Agent 健康上报", ui: "Agent 列表/日志平台", auth: false },
    ],
  },
  {
    title: "项目管理",
    routes: [
      { method: "GET", path: "/api/v1/projects", summary: "项目分页列表", ui: "项目管理", auth: true },
      { method: "POST", path: "/api/v1/projects", summary: "创建项目", ui: "项目管理", auth: true },
      { method: "PUT", path: "/api/v1/projects/:id", summary: "更新项目", ui: "项目管理", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id", summary: "删除项目", ui: "项目管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/members", summary: "项目成员列表", ui: "项目成员", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/members", summary: "添加项目成员", ui: "项目成员", auth: true },
      { method: "PUT", path: "/api/v1/projects/:id/members/:memberId", summary: "更新项目成员角色", ui: "项目成员", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id/members/:memberId", summary: "移除项目成员", ui: "项目成员", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/server-groups/tree", summary: "服务器分组树", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/server-groups", summary: "创建/更新服务器分组", ui: "服务器管理", auth: true },
      { method: "PUT", path: "/api/v1/projects/:id/server-groups/:groupId", summary: "更新服务器分组", ui: "服务器管理", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id/server-groups/:groupId", summary: "删除服务器分组", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/servers", summary: "服务器分页列表", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers", summary: "创建/更新服务器", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/servers/:serverId", summary: "服务器详情", ui: "服务器管理/操作台", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id/servers/:serverId", summary: "删除服务器", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers/:serverId/exec", summary: "执行服务器命令", ui: "服务器操作台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/servers/:serverId/terminal/ws", summary: "交互式终端 WebSocket", ui: "服务器操作台", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers/test", summary: "单机连通性测试", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers/test/batch", summary: "批量连通性测试", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers/sync", summary: "同步服务器状态", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/servers/import", summary: "导入服务器", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/servers/import-template", summary: "下载服务器导入模板", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/servers/export", summary: "导出服务器", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/cloud-accounts", summary: "云账号列表", ui: "服务器管理", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/cloud-accounts", summary: "创建云账号", ui: "服务器管理", auth: true },
      { method: "PUT", path: "/api/v1/projects/:id/cloud-accounts/:accountId", summary: "更新云账号", ui: "服务器管理", auth: true },
      { method: "PUT", path: "/api/v1/projects/:id/cloud-accounts/:accountId/sync", summary: "同步云实例", ui: "服务器管理", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/services", summary: "项目服务列表", ui: "服务配置", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/services", summary: "创建/更新项目服务", ui: "服务配置", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id/services/:serviceId", summary: "删除项目服务", ui: "服务配置", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/log-sources", summary: "日志源列表", ui: "日志源配置", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/log-sources", summary: "创建/更新日志源", ui: "日志源配置", auth: true },
      { method: "DELETE", path: "/api/v1/projects/:id/log-sources/:logSourceId", summary: "删除日志源", ui: "日志源配置", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/agents/list", summary: "Agent 列表", ui: "Agent 列表", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/agents/heartbeat-refresh", summary: "批量刷新 Agent 心跳", ui: "Agent 列表", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/agents/status", summary: "Agent 状态", ui: "日志平台", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/agents/bootstrap", summary: "Agent 引导", ui: "日志平台", auth: true },
      { method: "POST", path: "/api/v1/projects/:id/agents/rotate-token", summary: "Agent 轮转 Token", ui: "日志平台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/agents/discovery", summary: "Agent 自动发现", ui: "日志平台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/logs/stream", summary: "日志流", ui: "日志平台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/logs/export", summary: "导出日志", ui: "日志平台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/log-files", summary: "远端日志文件列表", ui: "日志平台", auth: true },
      { method: "GET", path: "/api/v1/projects/:id/log-units", summary: "远端 systemd 单元列表", ui: "日志平台", auth: true },
    ],
  },
];
