# 菜单需求：CR 实例管理（`/crs`）

## 1. 定位

- **路由**：`/crs`，`crs-page`。  
- **目标**：按资源类型浏览 **自定义资源实例**，支持详情、YAML、删除。

## 2. API（`/api/v1/crs`）

- `GET /resources` 类型目录  
- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 不同 CRD 的 schema 差异大；应用 YAML 前需通过 `kubectl`/openapi 校验。
