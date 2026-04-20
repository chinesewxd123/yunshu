# 菜单需求：Ingress 管理（`/ingresses`）

## 1. 定位

- **路由**：`/ingresses`，`ingresses-page`。  
- **目标**：Ingress 列表、详情、YAML、删除，以及 **重启 Ingress-Nginx Controller Pod**（证书/配置热更新场景）。

## 2. API（`/api/v1/ingresses`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  
- `POST /nginx/restart`：滚动重启控制器（慎用）。  

## 3. 注意事项

- 重启 Nginx 可能造成 briefly 流量抖动；与证书轮转流程配合。
