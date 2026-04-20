# 菜单需求：Secret 管理（`/secrets`）

## 1. 定位

- **路由**：`/secrets`，`secrets-page`。  
- **目标**：Secret 列表、详情、YAML 应用、删除。

## 2. API（`/api/v1/secrets`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- **敏感数据**：列表与详情展示应 base64/脱敏（以前端为准）；审计与导出需额外控制。
