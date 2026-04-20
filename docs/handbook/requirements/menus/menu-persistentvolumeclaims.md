# 菜单需求：PersistentVolumeClaim（`/persistentvolumeclaims`）

## 1. 定位

- **路由**：`/persistentvolumeclaims`，`persistentvolumeclaims-page`。  
- **目标**：命名空间内 PVC 列表、详情、YAML、删除。

## 2. API（`/api/v1/persistentvolumeclaims`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 与 StatefulSet/数据库等有状态负载强相关；删除前确认数据备份。
