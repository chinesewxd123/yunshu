# 需求说明：Kubernetes 控制台

## 1. 目标

对接已导入的 **K8s 集群**，提供命名空间、工作负载、Pod、网络、存储、RBAC、CRD/CR、事件等资源的列表、详情、YAML 应用与常用运维操作。

## 2. 功能模块（菜单对应）

- 集群管理、命名空间、Node、组件状态、Pod、Deployment/StatefulSet/DaemonSet/CronJob/Job  
- ConfigMap/Secret/Service/Ingress/PV/PVC/StorageClass  
- RBAC（Role/RoleBinding/ClusterRole/ClusterRoleBinding）  
- CRD、自定义资源 CR  
- 事件

## 3. 权限与注意事项

- **API 权限**：`cmd/seed.go` 中为各 REST 路径配置了 GET/POST/DELETE 等。
- **K8s 三元策略**：在 API 权限不足时，部分 **GET** 请求若角色已下发 `k8s:cluster:{id}:...` 策略，可读对应路径（见权限手册）。
- 集群 kubeconfig 存于 `k8s_clusters`，连接失败时页面展示连接状态。

## 4. 数据

主表：`k8s_clusters`；资源实体在集群侧，平台不落业务表（除集群元数据）。

1. 为什么「只读」还能进 Pod Exec？
根因不是「白名单」，而是 档位计算把 GET /api/v1/pods/exec/ws 当成了普通只读接口：

expandReadonly 会扫描权限表里所有 GET 且 IsK8sReadAPIPath 为真 的条目；
IsK8sReadAPIPath 用 /api/v1/pods 前缀 判断，于是 /api/v1/pods/exec/ws 也被算进「只读展开」；
RequiredK8sAccessRank 先命中「只读展开」→ 认为这条路由 只要 readonly（1） 就够；
你给用户的是 只读档，EffectiveTier 也是 1 → 中间件认为档位足够，Exec 就放进去了。
已在 internal/service/k8s_cluster_preset.go 的 expandReadonly 里 排除路径中含 exec 的 GET（终端/WebSocket 不应参与 readonly 档位展开）。编译已通过。

请 重新编译并部署后端；部署后：仅 readonly 档位应无法再建立 Exec/WebSocket（仍需 readonly_exec）。若仍能通过，再查是否还有 readonly_exec / admin 档位（例如另一条主体或「全部集群」档位叠加取高）。

2. 命名空间列表「是不是只显示白名单」？
逻辑是：

没有任何「命名空间白名单」规则（k8s_namespace_allow_rules）作用于当前用户在该集群 → 列表 = 集群里全部 NS，再减去黑名单（kube-system、monitoring 等不应出现）。
只要对该用户任一主体在该集群 存在任意一条白名单 → 进入 「仅允许白名单内 NS」 模式 → 列表里 只会出现白名单里的 NS（你只看到 default，多半是 只给 default 写了白名单，而不是「黑名单=白名单」）。
所以：不是「有黑名单就只显示白名单」；只有配置了白名单才会变成「只显示白名单里的空间」。