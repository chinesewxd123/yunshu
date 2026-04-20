# 菜单需求：服务配置（`/project-services`）

## 1. 定位

- **路由**：`/project-services`，`project-services-page`。  
- **目标**：登记项目内**业务服务**元数据（名称、端口、描述等），常与日志、监控展示关联。

## 2. API

- `GET /api/v1/projects/:id/services`  
- `POST /api/v1/projects/:id/services`（创建或更新，以后端为准）  
- `DELETE /api/v1/projects/:id/services/:serviceId`

## 3. 数据表

- `services`（项目维度）。

## 4. 注意事项

- 与 K8s **Service** 资源不同名；此处为 CMDB 业务服务模型。
