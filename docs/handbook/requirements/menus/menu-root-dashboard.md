# 菜单需求：资产总览（`/`）

## 1. 定位与入口

- **路由**：`/`（React `Route index` → `DashboardPage`）。  
- **种子菜单**：名称为「资产总览」，无 `component` 字段，与固定首页一致。  
- **目标用户**：已登录管理员/运维，用于一眼查看平台规模与健康趋势。

## 2. 页面功能

| 功能块 | 说明 |
|--------|------|
| 系统健康 | 调用 `GET /api/v1/health`（或封装在 `getHealth`），展示服务名、环境、版本等。 |
| 资产统计卡片 | 调用 `GET /api/v1/overview`，聚合用户总数、集群数、待审核注册、项目服务器数、Pod 正常/异常、事件统计等。 |
| 趋势图 | `GET /api/v1/overview/trends`，折线展示历史趋势（具体维度以前端 `overview` 服务为准）。 |
| 最近操作/登录 | 可选拉取 `operation-logs`、`login-logs` 列表若干条，用于审计入口跳转。 |

## 3. 依赖 API（摘要）

- `GET /api/v1/health`  
- `GET /api/v1/overview`  
- `GET /api/v1/overview/trends`  
- （可选）`GET /api/v1/operation-logs`、`GET /api/v1/login-logs`  

完整定义见 OpenAPI 生成文件。

## 4. 数据与表

- 概览数据由 `overview` 服务聚合多表（用户、集群、注册申请、服务器、K8s 缓存等），**非单表 CRUD 页**。

## 5. 权限与注意事项

- 需登录；接口走 Casbin `Authorize`。  
- **无集群权限用户**：若仅开放概览读权限，需确保 `overview` 相关 `GET` 已在种子权限中并对角色授权。  
- 大租户下卡片数字为**近似实时**，强一致统计需单独报表。

## 6. 异常与排障

- 卡片全 0：检查 DB 连接、K8s 集群是否注册、Redis 是否可用。  
- 趋势图为空：检查 `trends` 接口时间范围与后端聚合逻辑。
