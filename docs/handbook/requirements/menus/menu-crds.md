# 菜单需求：CRD 管理（`/crds`）

## 1. 定位

- **路由**：`/crds`，`crds-page`。  
- **目标**：查看集群中 **CustomResourceDefinition** 列表、详情、YAML 应用、删除。

## 2. API（`/api/v1/crds`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 删除 CRD 会级联删除对应 CR 实例；生产环境通常禁止随意删除。
