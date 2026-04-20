# 菜单需求：PersistentVolume（`/persistentvolumes`）

## 1. 定位

- **路由**：`/persistentvolumes`，`persistentvolumes-page`。  
- **目标**：集群级别 PV 列表、详情、YAML、删除。

## 2. API（`/api/v1/persistentvolumes`）

- `GET`、`GET /detail`、`POST /apply`、`DELETE`  

## 3. 注意事项

- 删除 PV 可能影响仍有 PVC 绑定的工作负载；需先释放存储。
