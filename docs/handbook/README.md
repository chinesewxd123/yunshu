# 产品手册（YunShu CMDB / go-permission-system）

本目录为与源码同步维护的**需求、数据库、接口、权限**说明，配合根目录 `README.md` 与 `docs/deployment/` 下的部署文档使用。

## 文档索引

| 文档 | 说明 |
|------|------|
| [00-architecture-analysis-and-optimization.md](./00-architecture-analysis-and-optimization.md) | 功能域梳理、调用链、SQL/性能与可维护性优化建议 |
| [requirements/](./requirements/) | 按业务域拆分的需求说明（功能、子功能、注意事项） |
| [requirements/menus/_INDEX.md](./requirements/menus/_INDEX.md) | **按菜单（路由）一页一文**的详细需求，含 API/表/注意事项 |
| [requirements/menus/menu-k8s-resource-pattern.md](./requirements/menus/menu-k8s-resource-pattern.md) | K8s 控制台列表类页面通用模式（集群/三元策略/典型能力） |
| [database/schema-and-relationships.md](./database/schema-and-relationships.md) | 表清单、关系说明、与 GORM 模型对应 |
| [api/http-api-conventions.md](./api/http-api-conventions.md) | 统一响应、分页、鉴权头、错误码；与 Swagger 关系 |
| [permissions/casbin-and-k8s-triple-policy.md](./permissions/casbin-and-k8s-triple-policy.md) | API 级 Casbin、K8s 三元策略、中间件行为 |

## 机器可读接口定义

- OpenAPI（**由路由全量生成，推荐以此为准**）：`docs/apipost/permission-system.openapi.yaml`  
  - 重新生成：`go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml`
- Swagger 生成物：`docs/swagger/swagger.yaml`、`docs/swagger/swagger.json`（以实际构建为准）

## 相关文档

- 告警通知：`docs/alert-notify-guide.md`
- 日志平台 gRPC：`docs/log-platform-api.md`
- **麒麟 V10 部署**：`docs/deployment/KYLIN_V10_X86_64.md`
