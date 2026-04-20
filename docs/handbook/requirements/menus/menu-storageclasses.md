# 菜单需求：StorageClass（`/storageclasses`）

## 1. 定位

- **路由**：`/storageclasses`，`storageclasses-page`。  
- **目标**：存储类列表、详情、YAML、删除。

## 2. API（`/api/v1/storageclasses`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 存储类影响动态供给与默认 reclaim 策略；删除前确认无 PVC 依赖。
