# Yunshu

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![Ant Design](https://img.shields.io/badge/Ant%20Design-5.x-1677ff?style=flat-square&logo=antdesign)](https://ant.design/)
[![Status](https://img.shields.io/badge/Project-Active-brightgreen?style=flat-square)](#)

> 基于 Go + React 的 Kubernetes 运维与项目化告警平台，涵盖系统管理、权限管理、项目管理、K8s 资源管理、告警平台与日志平台。

---

## 目录

- [项目简介](#项目简介)
- [快速开始](#快速开始)
- [功能状态标记说明](#功能状态标记说明)
- [页面功能与截图](#页面功能与截图)
  - [1. 登录与概览](#1-登录与概览)
  - [2. 系统管理](#2-系统管理)
  - [3. 项目管理](#3-项目管理)
  - [4. 告警平台](#4-告警平台)
  - [5. Kubernetes 管理](#5-kubernetes-管理)
- [告警通知与恢复示例](#告警通知与恢复示例)
- [数据库 ER 图](#数据库-er-图)
- [项目结构](#项目结构)
- [文档链接](#文档链接)
- [参考项目](#参考项目)

---

## 项目简介

Yunshu 主要能力：

- 多模块后台管理（用户、角色、菜单、组织、字典、审计）
- Casbin 权限体系与 K8s 三元策略限制
- 项目维度的服务器、服务、日志源、Agent、日志平台
- 告警数据源、告警规则、值班、静默、策略与告警记录
- Kubernetes 常见资源可视化管理（工作负载、网络、存储、RBAC、CRD）

---

## 快速开始

### 环境要求

- Go 1.23+
- Node.js 18+
- MySQL
- Redis

### 启动步骤

```bash
git clone <your-repo-url>
cd yunshu

go mod download
cd web && npm install && cd ..

go run . migrate
go run . seed
go run . server
```

新终端启动前端：

```bash
cd web
npm run dev
```

---

## 功能状态标记说明

- ![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square) 已实现：`- [x]`
- ![todo](https://img.shields.io/badge/状态-未实现-ef4444?style=flat-square) 未实现/待规划：`- [ ]`

---

## 页面功能与截图

> 按你 `images` 目录中的页面分组整理，每个页面给出“已实现/待规划”能力点。

### 1. 登录与概览

#### 系统登录页面-账密登录
![系统登录页面-账密登录](./images/系统登录页面-账密登录.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 用户名/密码登录
- [x] 登录失败提示与鉴权校验
- [ ] 第三方统一登录（如 OAuth2 / SSO）

#### 系统登录页面-邮箱登录
![系统登录页面-邮箱登录](./images/系统登录页面-邮箱登录.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 邮箱验证码登录流程
- [x] 登录后权限菜单动态加载
- [ ] 多因子验证（MFA）统一入口

#### 概览页面
![概览页面](./images/概览页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 系统总览数据展示
- [x] 关键指标可视化
- [ ] 指标自定义看板

---

### 2. 系统管理

#### 系统管理-用户管理页面
![系统管理-用户管理页面](./images/系统管理-用户管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 用户增删改查
- [x] 用户状态管理
- [ ] 批量导入审批流

#### 用户管理-用户设置页面
![用户管理-用户设置页面](./images/用户管理-用户设置页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 个人信息维护
- [x] 账号基础设置
- [ ] 个性化主题/通知偏好

#### 系统管理-角色管理页面
![系统管理-角色管理页面](./images/系统管理-角色管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 角色增删改查
- [x] 角色与用户绑定
- [ ] 角色模板快速复制

#### 系统管理-授权管理页面
![系统管理-授权管理页面](./images/系统管理-授权管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] Casbin 权限规则维护
- [x] API 级授权分配
- [ ] 可视化权限冲突分析

#### 系统管理-菜单管理页面
![系统管理-菜单管理页面](./images/系统管理-菜单管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 菜单树管理
- [x] 菜单顺序与父子层级维护
- [ ] 菜单版本回滚

#### 系统管理-组织架构管理页面
![系统管理-组织架构管理页面](./images/系统管理-组织架构管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 部门树管理
- [x] 组织层级调整
- [ ] 组织历史变更审计报表

#### 系统管理-数据字典管理页面
![系统管理-数据字典管理页面](./images/系统管理-数据字典管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 字典项增删改查
- [x] 字典在业务表单中复用
- [ ] 字典国际化多语言

#### 系统管理-登录日志页面
![系统管理-登录日志页面](./images/系统管理-登录日志页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 登录记录查询
- [x] 关键字段筛选
- [ ] 异常登录自动告警

#### 系统管理-操作日志管理
![系统管理-操作日志管理](./images/系统管理-操作日志管理.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 操作行为审计
- [x] 接口请求与操作者关联
- [ ] 审计日志归档到对象存储

#### 系统管理-IP封禁管理页面
![系统管理-IP封禁管理页面](./images/系统管理-IP封禁管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 封禁列表管理
- [x] 封禁状态即时生效
- [ ] 自动解封策略配置

#### 系统管理-注册审核管理页面
![系统管理-注册审核管理页面](./images/系统管理-注册审核管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 注册申请审核
- [x] 审核状态流转
- [ ] 审核 SLA 超时提醒

#### 系统管理-API管理页面
![系统管理-API管理页面](./images/系统管理-API管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] API 资源项管理
- [x] API 与权限点绑定
- [ ] API 文档自动回填

#### 系统管理-页面切换功能
![系统管理-页面切换功能](./images/系统管理-页面切换功能.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 菜单路由切换
- [x] 多页面导航
- [ ] 最近访问页签固定功能

---

### 3. 项目管理

#### 项目管理-项目列表页面
![项目管理-项目列表页面](./images/项目管理-项目列表页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 项目增删改查
- [x] 项目成员入口
- [x] 操作栏样式统一优化
- [ ] 项目归档功能

#### 项目管理-服务器管理页面
![项目管理-服务器管理页面](./images/项目管理-服务器管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 项目服务器管理
- [x] 基础连接信息维护
- [ ] 服务器批量导入向导

#### 项目管理-服务配置页面
![项目管理-服务配置页面](./images/项目管理-服务配置页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 服务配置维护
- [x] 服务与项目/服务器关联
- [ ] 服务模板复用

#### 项目管理-日志源配置页面
![项目管理-日志源配置页面](./images/项目管理-日志源配置页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 日志源增删改查
- [x] 日志采集类型与路径配置
- [ ] 日志源连通性自检

#### 项目管理-agent列表管理页面
![项目管理-agent列表管理页面](./images/项目管理-agent列表管理页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] Agent 列表展示
- [x] Agent 状态查询
- [ ] Agent 分组与批量操作

#### 项目管理-日志平台页面
![项目管理-日志平台页面](./images/项目管理-日志平台页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] SSE 实时日志流
- [x] include/exclude/highlight 过滤
- [x] 文件级别筛选
- [ ] 日志收藏与分享

---

### 4. 告警平台

#### 告警平台-数据源配置页面
![告警平台-数据源配置页面](./images/告警平台-数据源配置页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 告警数据源按项目绑定
- [x] 数据源列表与筛选
- [ ] 数据源健康探测

#### 告警平台-告警规则与值班人配置页面
![告警平台-告警规则与值班人配置页面](./images/告警平台-告警规则与值班人配置页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 规则管理与值班人配置
- [x] 规则项目归属由数据源派生
- [ ] 规则变更审批流

#### 告警平台-值班总览页面
![告警平台-值班总览页面](./images/告警平台-值班总览页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 值班排班总览
- [x] 值班关联规则可视化
- [ ] 值班冲突自动检测

#### 告警平台-告警策略与告警记录页面
![告警平台-告警策略与告警记录页面](./images/告警平台-告警策略与告警记录页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 告警策略配置
- [x] 告警记录查询
- [ ] 记录导出与归档

#### 告警平台-告警静默页面
![告警平台-告警静默页面](./images/告警平台-告警静默页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 静默规则管理
- [x] 生效时间控制
- [ ] 静默模板管理

#### 告警通知-告警渠道页面
![告警通知-告警渠道页面](./images/告警通知-告警渠道页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 告警渠道配置
- [x] 多渠道参数维护
- [ ] 渠道联调测试按钮

#### 告警平台-promql查询页面
![告警平台-promql查询页面](./images/告警平台-promql查询页面.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] PromQL 查询调试
- [x] 查询结果展示
- [ ] 常用查询语句收藏

---

## 告警通知与恢复示例

#### 告警通知与恢复-钉钉示例
![告警通知与恢复-钉钉示例](./images/告警通知与恢复-钉钉示例.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 告警触发消息投递（钉钉渠道）
- [x] 告警恢复消息投递（Recover 通知）
- [x] 策略匹配结果与通知链路联动
- [ ] 钉钉消息模板可视化编辑器

#### 告警通知与恢复-邮箱示例
![告警通知与恢复-邮箱示例](./images/告警通知与恢复-邮箱示例.png)
![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 告警触发邮件通知
- [x] 告警恢复邮件通知
- [x] 处理人 + 项目成员邮箱合并去重
- [ ] 邮件模板分级管理（按策略/渠道）

---

### 5. Kubernetes 管理

#### 集群与基础资源

![k8s-集群管理页面](./images/k8s-集群管理页面.png)
![k8s-组件状态页面](./images/k8s-组件状态页面.png)
![k8s-命名空间管理页面](./images/k8s-命名空间管理页面.png)
![k8s-Node管理页面](./images/k8s-Node管理页面.png)
![k8s-Pod管理页面](./images/k8s-Pod管理页面.png)

![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 集群、组件状态、命名空间、节点、Pod 基础管理
- [x] Pod 详情改为只读，编辑收口到表单
- [ ] 多集群统一搜索

#### 工作负载

![k8s-Deployment管理页面](./images/k8s-Deployment管理页面.png)
![k8s-StatefulSet管理页面](./images/k8s-StatefulSet管理页面.png)
![k8s-DaemonSet管理页面](./images/k8s-DaemonSet管理页面.png)
![k8s-job管理页面](./images/k8s-job管理页面.png)
![k8s-Cronjob管理页面](./images/k8s-Cronjob管理页面.png)

![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] 工作负载列表与详情
- [x] 表单创建与编辑能力
- [ ] 工作负载版本回滚助手

#### 网络与配置

![k8s-Service管理页面](./images/k8s-Service管理页面.png)
![k8s-ingress管理页面](./images/k8s-ingress管理页面.png)
![k8s-IngressClass管理页面](./images/k8s-IngressClass管理页面.png)
![k8s-网络策略管理页面](./images/k8s-网络策略管理页面.png)
![k8s-configmap管理页面](./images/k8s-configmap管理页面.png)
![k9s-secret管理页面](./images/k9s-secret管理页面.png)

![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] Service/Ingress/NetworkPolicy 管理
- [x] ConfigMap/Secret 管理
- [ ] Ingress 联调诊断向导

#### 存储与扩展资源

![k8s-PV管理页面](./images/k8s-PV管理页面.png)
![k8s-PVC管理页面](./images/k8s-PVC管理页面.png)
![k8s-storageclass管理页面](./images/k8s-storageclass管理页面.png)
![k8s-CRD管理页面](./images/k8s-CRD管理页面.png)
![k8s-events管理页面](./images/k8s-events管理页面.png)

![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] PV/PVC/StorageClass 管理
- [x] CRD 与事件管理
- [ ] CR 模板库

#### RBAC 与三元策略

![k8s-role管理页面](./images/k8s-role管理页面.png)
![k8s-rolebinding管理页面](./images/k8s-rolebinding管理页面.png)
![k8s-clusterrole管理页面](./images/k8s-clusterrole管理页面.png)
![k8s-clusterrolebinding管理页面](./images/k8s-clusterrolebinding管理页面.png)
![k8s-三元策略限制页面](./images/k8s-三元策略限制页面.png)

![done](https://img.shields.io/badge/状态-已实现-22c55e?style=flat-square)
- [x] K8s RBAC 资源可视化管理
- [x] 三元策略限制（cluster + namespace + action）
- [ ] 三元策略模拟器（预检查）

---

## 数据库 ER 图

> 当前默认数据库名（见 `configs/config.yaml`）：`permission_system`。  
> README 仅保留总览图；5 大域精细版请查看：`docs/handbook/database/er-diagrams.md`。

```mermaid
erDiagram
  USERS ||--o{ USER_ROLES : "user_id"
  ROLES ||--o{ USER_ROLES : "role_id"
  DEPARTMENTS ||--o{ USERS : "department_id"
  USERS ||--o{ REGISTRATION_REQUESTS : "reviewer_id"
  USERS ||--o{ LOGIN_LOGS : "user_id(nullable)"
  USERS ||--o{ OPERATION_LOGS : "user_id"
  USERS ||--o{ ALERT_SILENCES : "created_by"

  PROJECTS ||--o{ PROJECT_MEMBERS : "project_id"
  USERS ||--o{ PROJECT_MEMBERS : "user_id"

  PROJECTS ||--o{ SERVER_GROUPS : "project_id"
  SERVER_GROUPS ||--o{ SERVER_GROUPS : "parent_id"
  PROJECTS ||--o{ SERVERS : "project_id"
  SERVER_GROUPS ||--o{ SERVERS : "group_id(nullable)"
  SERVERS ||--|| SERVER_CREDENTIALS : "server_id(unique)"
  SERVERS ||--o{ SERVICES : "server_id"
  SERVICES ||--o{ SERVICE_LOG_SOURCES : "service_id"
  PROJECTS ||--o{ CLOUD_ACCOUNTS : "project_id"
  SERVER_GROUPS ||--o{ CLOUD_ACCOUNTS : "group_id"

  PROJECTS ||--o{ LOG_AGENTS : "project_id"
  SERVERS ||--o{ LOG_AGENTS : "server_id(unique)"
  PROJECTS ||--o{ AGENT_DISCOVERIES : "project_id"
  SERVERS ||--o{ AGENT_DISCOVERIES : "server_id"

  PROJECTS ||--o{ ALERT_DATASOURCES : "project_id"
  ALERT_DATASOURCES ||--o{ ALERT_MONITOR_RULES : "datasource_id"
  ALERT_MONITOR_RULES ||--o{ ALERT_RULE_ASSIGNEES : "monitor_rule_id"
  ALERT_MONITOR_RULES ||--o{ ALERT_DUTY_BLOCKS : "monitor_rule_id"
  ALERT_CHANNELS ||--o{ ALERT_EVENTS : "channel_id"
```

更多细分图（系统管理 / 项目管理 / 告警 / 日志 / K8s）请见：`docs/handbook/database/er-diagrams.md`。

---

## 项目结构

```text
yunshu/
├── cmd/          # 命令入口（server/migrate/seed/logagent）
├── configs/      # 配置文件
├── docs/         # 产品手册与部署文档
├── images/       # 页面截图与 README 展示图
├── internal/     # 后端核心代码
└── web/          # 前端 React 应用
```

---

## 文档链接

- 产品手册：`docs/handbook/README.md`
- 部署文档：`docs/deployment/KYLIN_V10_X86_64.md`
- 告警通知说明：`docs/alert-notify-guide.md`

---

## 参考项目

- [dnsjia/luban](https://github.com/dnsjia/luban)

