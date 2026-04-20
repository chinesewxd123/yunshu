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
