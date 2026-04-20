# 菜单需求：Service 管理（`/k8s-services`）

## 1. 定位

- **路由**：`/k8s-services`，`k8s-services-page`。  
- **目标**：Kubernetes **Service** 资源管理（勿与「项目-服务配置」混淆，后者为 CMDB `services` 表）。

## 2. API（`/api/v1/k8s-services`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- ClusterIP/NodePort/LoadBalancer 变更会影响访问入口；与 Ingress 配合时需统一规划。
