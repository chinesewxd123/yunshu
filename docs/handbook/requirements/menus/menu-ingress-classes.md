# 菜单需求：IngressClass（`/ingress-classes`）

## 1. 定位

- **路由**：`/ingress-classes`，`ingress-classes-page`。  
- **目标**：IngressClass 列表、详情、YAML、删除。

## 2. API（`/api/v1/ingresses/classes`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`（路径以前端封装为准，后端见 router `ingresses` 子路径）。

## 3. 注意事项

- Ingress 资源需引用正确的 `ingressClassName`。
