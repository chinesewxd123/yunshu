# 菜单需求：命名空间管理（`/namespaces`）

## 1. 定位

- **路由**：`/namespaces`，`namespaces-page`。  
- **目标**：在选定集群下列出 **Namespace**，支持详情、YAML 应用、删除。

## 2. API（`/api/v1/namespaces`）

- `GET` 列表，`GET /detail` 详情  
- `POST /apply` 应用 YAML  
- `DELETE` 删除  

## 3. 权限

- `k8sScopeAuthorize` + Casbin；读接口可能受三元策略兜底。

## 4. 注意事项

- 删除命名空间会级联删除其内资源，需强确认。
