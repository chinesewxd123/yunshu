# 需求说明文档索引

每个文件对应一个业务域，包含**目标用户、功能列表、子功能、注意事项、与相关表/接口的对应关系**。

| 文件 | 域 |
|------|-----|
| [R-01-auth-and-identity.md](./R-01-auth-and-identity.md) | 登录、注册申请、个人设置、JWT |
| [R-02-project-management.md](./R-02-project-management.md) | 项目、成员、服务器、服务、日志源、云账号 |
| [R-03-alert-and-monitor.md](./R-03-alert-and-monitor.md) | 告警通道、策略、数据源、静默、监控规则、处理人、值班 |
| [R-04-kubernetes-console.md](./R-04-kubernetes-console.md) | 集群、工作负载、网络存储、RBAC、事件等 |
| [R-05-system-administration.md](./R-05-system-administration.md) | 用户/角色/部门、权限、策略、菜单、字典、日志审计、封禁 IP |
| [R-06-log-platform-and-agent.md](./R-06-log-platform-and-agent.md) | Log Agent、gRPC、发现、项目日志 |

## 按菜单（前端路由）细分

每个可见菜单对应一篇说明（路由、组件、API、表、权限、注意点）：**[menus/_INDEX.md](./menus/_INDEX.md)**。  
Kubernetes 多页共用模式见 **[menus/menu-k8s-resource-pattern.md](./menus/menu-k8s-resource-pattern.md)**。
