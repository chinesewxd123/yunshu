# 菜单需求：K8s 三元策略（`/k8s-scoped-policies`）

## 1. 定位

- **路由**：`/k8s-scoped-policies`，`k8s-scoped-policies-page`。  
- **目标**：为**角色**下发 **集群 + 命名空间 + API 路径** 维度的细粒度权限（Casbin 对象前缀 `k8s:cluster:...`）。

## 2. 功能清单

| 功能 | 说明 |
|------|------|
| 动作/路径目录 | `GET /api/v1/k8s-policies/actions`、`GET /api/v1/k8s-policies/paths`。 |
| 按角色列已有策略 | `GET /api/v1/k8s-policies`。 |
| 下发 | `POST /api/v1/k8s-policies/grant`。 |

## 3. 与 API 权限关系

- 见 `docs/handbook/permissions/casbin-and-k8s-triple-policy.md`：**GET** 部分 K8s 读接口在具备三元策略时可放行。  
- **写操作**仍以实际 Casbin 模型与策略为准。

## 4. 注意事项

- 策略变更后用户需重新加载权限或重新登录（以同步实现为准）。  
- 勿与「授权管理」中 API 权限混淆：二者互补。
