package cmd

import (
	"context"
	"errors"
	"fmt"

	"go-permission-system/internal/bootstrap"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/service"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed default admin user, roles and permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.NewBuilder().
			WithConfig(configPath).
			WithLogger().
			WithMySQL().
			WithDictOverrides().
			WithCasbin().
			Build()
		if err != nil {
			return err
		}
		defer app.Close()

		ctx := context.Background()
		// 确保新增字段（如 permissions.k8s_scope_enabled）在 seed 前已完成迁移
		if err := bootstrap.AutoMigrateModels(app.DB); err != nil {
			return err
		}

		// 历史错误：撤销策略实际路由为 DELETE /api/v1/policies（JSON body），与 /api/v1/policies/:id 不匹配，会导致无法撤销授权
		if err := service.RemovePermissionPolicies(app.Enforcer, "/api/v1/policies/:id", "DELETE"); err != nil {
			return err
		}
		if err := app.DB.WithContext(ctx).Where("resource = ? AND action = ?", "/api/v1/policies/:id", "DELETE").Delete(&model.Permission{}).Error; err != nil {
			return err
		}

		permissions := defaultPermissions()
		for _, item := range permissions {
			var permission model.Permission
			err := app.DB.WithContext(ctx).
				Where("resource = ? AND action = ?", item.Resource, item.Action).
				First(&permission).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err = app.DB.WithContext(ctx).Create(&item).Error; err != nil {
					return err
				}
				permission = item
			} else if err != nil {
				return err
			} else {
				permission.Name = item.Name
				permission.Description = item.Description
				if err = app.DB.WithContext(ctx).Save(&permission).Error; err != nil {
					return err
				}
			}

			if _, err = app.Enforcer.AddPolicy("super-admin", permission.Resource, permission.Action); err != nil {
				return err
			}
		}

		adminRole := model.Role{
			Name:        "Super Admin",
			Code:        "super-admin",
			Description: "Built-in administrator role with full access.",
			Status:      model.StatusEnabled,
		}
		if err := upsertRole(ctx, app.DB, &adminRole); err != nil {
			return err
		}

		hashedPassword, err := password.Hash("Admin@123")
		if err != nil {
			return err
		}

		adminEmail := "rootwxd@163.com"
		adminUser := model.User{
			Username: "admin",
			Email:    &adminEmail,
			Password: hashedPassword,
			Nickname: "System Admin",
			Status:   model.StatusEnabled,
		}
		if err := upsertUser(ctx, app.DB, &adminUser); err != nil {
			return err
		}

		if err := app.DB.WithContext(ctx).Model(&adminUser).Association("Roles").Replace([]model.Role{adminRole}); err != nil {
			return err
		}
		if err := service.SyncUserRoles(app.Enforcer, adminUser.ID, []model.Role{adminRole}); err != nil {
			return err
		}

		if err := seedMenus(ctx, app.DB); err != nil {
			return err
		}

		app.Logger.Info.Info("seed completed", "username", adminUser.Username, "email", adminUser.Email, "password", "Admin@123")
		fmt.Println("seed completed: admin / Admin@123 / admin@example.com")
		return nil
	},
}

func upsertRole(ctx context.Context, db *gorm.DB, role *model.Role) error {
	var existing model.Role
	err := db.WithContext(ctx).Where("code = ?", role.Code).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(role).Error
	}
	if err != nil {
		return err
	}

	existing.Name = role.Name
	existing.Description = role.Description
	existing.Status = role.Status
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*role = existing
	return nil
}

func upsertUser(ctx context.Context, db *gorm.DB, user *model.User) error {
	var existing model.User
	err := db.WithContext(ctx).Where("username = ?", user.Username).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(user).Error
	}
	if err != nil {
		return err
	}

	existing.Email = user.Email
	existing.Password = user.Password
	existing.Nickname = user.Nickname
	existing.Status = user.Status
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*user = existing
	return nil
}

func defaultPermissions() []model.Permission {
	return []model.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET", Description: "View user list"},
		{Name: "创建用户", Resource: "/api/v1/users", Action: "POST", Description: "Create user"},
		{Name: "用户详情", Resource: "/api/v1/users/:id", Action: "GET", Description: "View user detail"},
		{Name: "更新用户", Resource: "/api/v1/users/:id", Action: "PUT", Description: "Update user"},
		{Name: "删除用户", Resource: "/api/v1/users/:id", Action: "DELETE", Description: "Delete user"},
		{Name: "分配用户角色", Resource: "/api/v1/users/:id/roles", Action: "PUT", Description: "Assign roles to user"},
		{Name: "导出用户", Resource: "/api/v1/users/export", Action: "GET", Description: "Export users to Excel"},
		{Name: "导入用户", Resource: "/api/v1/users/import", Action: "POST", Description: "Import users from Excel"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET", Description: "View role list"},
		{Name: "创建角色", Resource: "/api/v1/roles", Action: "POST", Description: "Create role"},
		{Name: "角色详情", Resource: "/api/v1/roles/:id", Action: "GET", Description: "View role detail"},
		{Name: "更新角色", Resource: "/api/v1/roles/:id", Action: "PUT", Description: "Update role"},
		{Name: "删除角色", Resource: "/api/v1/roles/:id", Action: "DELETE", Description: "Delete role"},
		{Name: "API列表", Resource: "/api/v1/permissions", Action: "GET", Description: "View permission list"},
		{Name: "创建API", Resource: "/api/v1/permissions", Action: "POST", Description: "Create permission"},
		{Name: "API详情", Resource: "/api/v1/permissions/:id", Action: "GET", Description: "View permission detail"},
		{Name: "更新API", Resource: "/api/v1/permissions/:id", Action: "PUT", Description: "Update permission"},
		{Name: "删除API", Resource: "/api/v1/permissions/:id", Action: "DELETE", Description: "Delete permission"},
		{Name: "授权列表", Resource: "/api/v1/policies", Action: "GET", Description: "View policy list"},
		{Name: "创建授权策略", Resource: "/api/v1/policies", Action: "POST", Description: "Grant permission to role"},
		{Name: "删除授权策略", Resource: "/api/v1/policies", Action: "DELETE", Description: "Revoke permission from role (JSON body)"},
		{Name: "K8s 三元策略动作目录", Resource: "/api/v1/k8s-policies/actions", Action: "GET", Description: "List k8s scoped action codes"},
		{Name: "K8s 三元策略路径目录", Resource: "/api/v1/k8s-policies/paths", Action: "GET", Description: "List k8s scoped paths"},
		{Name: "K8s 三元策略列表", Resource: "/api/v1/k8s-policies", Action: "GET", Description: "List k8s scoped policies by role"},
		{Name: "K8s 三元策略下发", Resource: "/api/v1/k8s-policies/grant", Action: "POST", Description: "Grant k8s scoped policies"},
		{Name: "注册审核列表", Resource: "/api/v1/registrations", Action: "GET", Description: "View registration requests"},
		{Name: "审核注册申请", Resource: "/api/v1/registrations/:id/review", Action: "POST", Description: "Review registration request"},
		{Name: "菜单树", Resource: "/api/v1/menus/tree", Action: "GET", Description: "View menu tree"},
		{Name: "创建菜单", Resource: "/api/v1/menus", Action: "POST", Description: "Create menu"},
		{Name: "更新菜单", Resource: "/api/v1/menus/:id", Action: "PUT", Description: "Update menu"},
		{Name: "删除菜单", Resource: "/api/v1/menus/:id", Action: "DELETE", Description: "Delete menu"},
		{Name: "告警通道列表", Resource: "/api/v1/alerts/channels", Action: "GET", Description: "List alert channels"},
		{Name: "创建告警通道", Resource: "/api/v1/alerts/channels", Action: "POST", Description: "Create alert channel"},
		{Name: "更新告警通道", Resource: "/api/v1/alerts/channels/:id", Action: "PUT", Description: "Update alert channel"},
		{Name: "删除告警通道", Resource: "/api/v1/alerts/channels/:id", Action: "DELETE", Description: "Delete alert channel"},
		{Name: "测试告警通道", Resource: "/api/v1/alerts/channels/:id/test", Action: "POST", Description: "Send test alert to channel"},
		{Name: "告警事件列表", Resource: "/api/v1/alerts/events", Action: "GET", Description: "List alert events"},
		{Name: "接收 Alertmanager Webhook", Resource: "/api/v1/alerts/webhook/alertmanager", Action: "POST", Description: "Receive alertmanager webhook"},
		{Name: "告警策略列表", Resource: "/api/v1/alerts/policies", Action: "GET", Description: "List alert policies"},
		{Name: "创建告警策略", Resource: "/api/v1/alerts/policies", Action: "POST", Description: "Create alert policy"},
		{Name: "更新告警策略", Resource: "/api/v1/alerts/policies/:id", Action: "PUT", Description: "Update alert policy"},
		{Name: "删除告警策略", Resource: "/api/v1/alerts/policies/:id", Action: "DELETE", Description: "Delete alert policy"},
		{Name: "告警数据源列表", Resource: "/api/v1/alerts/datasources", Action: "GET", Description: "List alert datasources"},
		{Name: "创建告警数据源", Resource: "/api/v1/alerts/datasources", Action: "POST", Description: "Create alert datasource"},
		{Name: "更新告警数据源", Resource: "/api/v1/alerts/datasources/:id", Action: "PUT", Description: "Update alert datasource"},
		{Name: "删除告警数据源", Resource: "/api/v1/alerts/datasources/:id", Action: "DELETE", Description: "Delete alert datasource"},
		{Name: "Prometheus 活跃告警快照", Resource: "/api/v1/alerts/datasources/:id/prometheus-alerts", Action: "GET", Description: "GET /api/v1/alerts proxy"},
		{Name: "PromQL 即时查询", Resource: "/api/v1/alerts/datasources/:id/query", Action: "POST", Description: "Prometheus instant query"},
		{Name: "PromQL 范围查询", Resource: "/api/v1/alerts/datasources/:id/query_range", Action: "POST", Description: "Prometheus range query"},
		{Name: "告警静默列表", Resource: "/api/v1/alerts/silences", Action: "GET", Description: "List alert silences"},
		{Name: "创建告警静默", Resource: "/api/v1/alerts/silences", Action: "POST", Description: "Create alert silence"},
		{Name: "批量创建告警静默", Resource: "/api/v1/alerts/silences/batch", Action: "POST", Description: "Batch create alert silences"},
		{Name: "更新告警静默", Resource: "/api/v1/alerts/silences/:id", Action: "PUT", Description: "Update alert silence"},
		{Name: "删除告警静默", Resource: "/api/v1/alerts/silences/:id", Action: "DELETE", Description: "Delete alert silence"},
		{Name: "监控告警规则列表", Resource: "/api/v1/alerts/monitor-rules", Action: "GET", Description: "List monitor alert rules"},
		{Name: "创建监控告警规则", Resource: "/api/v1/alerts/monitor-rules", Action: "POST", Description: "Create monitor alert rule"},
		{Name: "更新监控告警规则", Resource: "/api/v1/alerts/monitor-rules/:id", Action: "PUT", Description: "Update monitor alert rule"},
		{Name: "删除监控告警规则", Resource: "/api/v1/alerts/monitor-rules/:id", Action: "DELETE", Description: "Delete monitor alert rule"},
		{Name: "监控规则处理人", Resource: "/api/v1/alerts/monitor-rules/:id/assignees", Action: "GET", Description: "List rule assignees"},
		{Name: "配置监控规则处理人", Resource: "/api/v1/alerts/monitor-rules/:id/assignees", Action: "PUT", Description: "Upsert rule assignees"},
		{Name: "值班表列表", Resource: "/api/v1/alerts/duty-schedules", Action: "GET", Description: "List alert duty schedules"},
		{Name: "创建值班表", Resource: "/api/v1/alerts/duty-schedules", Action: "POST", Description: "Create alert duty schedule"},
		{Name: "更新值班表", Resource: "/api/v1/alerts/duty-schedules/:id", Action: "PUT", Description: "Update alert duty schedule"},
		{Name: "删除值班表", Resource: "/api/v1/alerts/duty-schedules/:id", Action: "DELETE", Description: "Delete alert duty schedule"},
		{Name: "值班班次列表", Resource: "/api/v1/alerts/duty-blocks", Action: "GET", Description: "List alert duty blocks"},
		{Name: "创建值班班次", Resource: "/api/v1/alerts/duty-blocks", Action: "POST", Description: "Create alert duty block"},
		{Name: "更新值班班次", Resource: "/api/v1/alerts/duty-blocks/:id", Action: "PUT", Description: "Update alert duty block"},
		{Name: "删除值班班次", Resource: "/api/v1/alerts/duty-blocks/:id", Action: "DELETE", Description: "Delete alert duty block"},
		{Name: "告警历史统计", Resource: "/api/v1/alerts/history/stats", Action: "GET", Description: "Alert history stats"},
		{Name: "登录日志列表", Resource: "/api/v1/login-logs", Action: "GET", Description: "View login logs"},
		{Name: "导出登录日志", Resource: "/api/v1/login-logs/export", Action: "GET", Description: "Export login logs to Excel"},
		{Name: "删除登录日志", Resource: "/api/v1/login-logs/:id", Action: "DELETE", Description: "Delete login log"},
		{Name: "批量删除登录日志", Resource: "/api/v1/login-logs/delete", Action: "POST", Description: "Batch delete login logs"},
		{Name: "操作历史列表", Resource: "/api/v1/operation-logs", Action: "GET", Description: "View operation logs"},
		{Name: "导出操作历史", Resource: "/api/v1/operation-logs/export", Action: "GET", Description: "Export operation logs to Excel"},
		{Name: "删除操作历史", Resource: "/api/v1/operation-logs/:id", Action: "DELETE", Description: "Delete operation log"},
		{Name: "批量删除操作历史", Resource: "/api/v1/operation-logs/delete", Action: "POST", Description: "Batch delete operation logs"},
		{Name: "查看封禁 IP 列表", Resource: "/api/v1/security/banned-ips", Action: "GET", Description: "View banned IPs list"},
		{Name: "解除封禁 IP", Resource: "/api/v1/security/banned-ips/unban", Action: "POST", Description: "Unban IP"},
		{Name: "资产总览", Resource: "/api/v1/overview", Action: "GET", Description: "Get system overview metrics"},
		{Name: "集群列表", Resource: "/api/v1/clusters", Action: "GET", Description: "View k8s clusters"},
		{Name: "创建集群", Resource: "/api/v1/clusters", Action: "POST", Description: "Create k8s cluster"},
		{Name: "更新集群", Resource: "/api/v1/clusters/:id", Action: "PUT", Description: "Update k8s cluster"},
		{Name: "删除集群", Resource: "/api/v1/clusters/:id", Action: "DELETE", Description: "Delete k8s cluster"},
		{Name: "启停集群", Resource: "/api/v1/clusters/:id/status", Action: "PUT", Description: "Enable/disable k8s cluster"},
		{Name: "集群连接状态", Resource: "/api/v1/clusters/:id/status", Action: "GET", Description: "Check k8s cluster status"},
		{Name: "集群命名空间", Resource: "/api/v1/clusters/:id/namespaces", Action: "GET", Description: "List cluster namespaces"},
		{Name: "组件状态列表", Resource: "/api/v1/clusters/:id/component-statuses", Action: "GET", Description: "List control plane component statuses"},
		{Name: "Pod 列表", Resource: "/api/v1/pods", Action: "GET", Description: "List pods"},
		{Name: "Pod 详情", Resource: "/api/v1/pods/detail", Action: "GET", Description: "Get pod detail"},
		{Name: "Pod 事件", Resource: "/api/v1/pods/events", Action: "GET", Description: "List pod events"},
		{Name: "Pod 日志", Resource: "/api/v1/pods/logs", Action: "GET", Description: "Get pod logs"},
		{Name: "Pod 日志下载", Resource: "/api/v1/pods/logs/download", Action: "GET", Description: "Download pod logs"},
		{Name: "Pod 日志流", Resource: "/api/v1/pods/logs/stream", Action: "GET", Description: "Stream pod logs"},
		{Name: "Pod 文件列表", Resource: "/api/v1/pods/files", Action: "GET", Description: "List pod files"},
		{Name: "Pod 文件读取", Resource: "/api/v1/pods/file", Action: "GET", Description: "Read pod file content"},
		{Name: "Pod 文件下载", Resource: "/api/v1/pods/file/download", Action: "GET", Description: "Download pod file"},
		{Name: "Pod 文件上传", Resource: "/api/v1/pods/file/upload", Action: "POST", Description: "Upload file to pod"},
		{Name: "Pod 文件删除", Resource: "/api/v1/pods/file/delete", Action: "POST", Description: "Delete pod file"},
		{Name: "Pod Exec", Resource: "/api/v1/pods/exec", Action: "POST", Description: "Exec command in pod"},
		{Name: "Pod 交互式终端", Resource: "/api/v1/pods/exec/ws", Action: "GET", Description: "Interactive exec terminal via websocket"},
		{Name: "Pod 重启", Resource: "/api/v1/pods/restart", Action: "POST", Description: "Restart pod"},
		{Name: "Pod YAML 创建", Resource: "/api/v1/pods/create/yaml", Action: "POST", Description: "Create pod by yaml"},
		{Name: "Pod 快捷创建", Resource: "/api/v1/pods/create/simple", Action: "POST", Description: "Create pod quickly"},
		{Name: "编辑并重建 Pod", Resource: "/api/v1/pods/update/simple", Action: "POST", Description: "Update pod by recreate"},
		{Name: "删除 Pod", Resource: "/api/v1/pods", Action: "DELETE", Description: "Delete pod"},
		{Name: "命名空间列表", Resource: "/api/v1/namespaces", Action: "GET", Description: "List namespaces"},
		{Name: "命名空间详情", Resource: "/api/v1/namespaces/detail", Action: "GET", Description: "Get namespace detail"},
		{Name: "命名空间应用 YAML", Resource: "/api/v1/namespaces/apply", Action: "POST", Description: "Apply namespace yaml"},
		{Name: "删除命名空间", Resource: "/api/v1/namespaces", Action: "DELETE", Description: "Delete namespace"},
		{Name: "Node 列表", Resource: "/api/v1/nodes", Action: "GET", Description: "List nodes"},
		{Name: "Node 详情", Resource: "/api/v1/nodes/detail", Action: "GET", Description: "Get node detail"},
		{Name: "Node 调度状态", Resource: "/api/v1/nodes/schedulability", Action: "POST", Description: "Cordon or uncordon node"},
		{Name: "Node 污点", Resource: "/api/v1/nodes/taints", Action: "PUT", Description: "Replace node taints"},

		{Name: "RBAC Role 列表", Resource: "/api/v1/rbac/roles", Action: "GET", Description: "List Roles"},
		{Name: "RBAC RoleBinding 列表", Resource: "/api/v1/rbac/rolebindings", Action: "GET", Description: "List RoleBindings"},
		{Name: "RBAC ClusterRole 列表", Resource: "/api/v1/rbac/clusterroles", Action: "GET", Description: "List ClusterRoles"},
		{Name: "RBAC ClusterRoleBinding 列表", Resource: "/api/v1/rbac/clusterrolebindings", Action: "GET", Description: "List ClusterRoleBindings"},
		{Name: "RBAC 详情", Resource: "/api/v1/rbac/detail", Action: "GET", Description: "Get RBAC detail"},
		{Name: "RBAC 应用 YAML", Resource: "/api/v1/rbac/apply", Action: "POST", Description: "Apply RBAC yaml"},
		{Name: "RBAC 删除", Resource: "/api/v1/rbac", Action: "DELETE", Description: "Delete RBAC resource"},

		{Name: "Deployment 列表", Resource: "/api/v1/deployments", Action: "GET", Description: "List deployments"},
		{Name: "Deployment 详情", Resource: "/api/v1/deployments/detail", Action: "GET", Description: "Get deployment detail"},
		{Name: "Deployment 应用 YAML", Resource: "/api/v1/deployments/apply", Action: "POST", Description: "Apply deployment yaml"},
		{Name: "Deployment 扩缩容", Resource: "/api/v1/deployments/scale", Action: "POST", Description: "Scale deployment"},
		{Name: "Deployment 重启", Resource: "/api/v1/deployments/restart", Action: "POST", Description: "Restart deployment"},
		{Name: "Deployment 关联 Pods", Resource: "/api/v1/deployments/pods", Action: "GET", Description: "List deployment related pods"},
		{Name: "删除 Deployment", Resource: "/api/v1/deployments", Action: "DELETE", Description: "Delete deployment"},

		{Name: "StatefulSet 列表", Resource: "/api/v1/statefulsets", Action: "GET", Description: "List statefulsets"},
		{Name: "StatefulSet 详情", Resource: "/api/v1/statefulsets/detail", Action: "GET", Description: "Get statefulset detail"},
		{Name: "StatefulSet 应用 YAML", Resource: "/api/v1/statefulsets/apply", Action: "POST", Description: "Apply statefulset yaml"},
		{Name: "StatefulSet 扩缩容", Resource: "/api/v1/statefulsets/scale", Action: "POST", Description: "Scale statefulset"},
		{Name: "StatefulSet 重启", Resource: "/api/v1/statefulsets/restart", Action: "POST", Description: "Restart statefulset"},
		{Name: "StatefulSet 关联 Pods", Resource: "/api/v1/statefulsets/pods", Action: "GET", Description: "List statefulset related pods"},
		{Name: "删除 StatefulSet", Resource: "/api/v1/statefulsets", Action: "DELETE", Description: "Delete statefulset"},

		{Name: "DaemonSet 列表", Resource: "/api/v1/daemonsets", Action: "GET", Description: "List daemonsets"},
		{Name: "DaemonSet 详情", Resource: "/api/v1/daemonsets/detail", Action: "GET", Description: "Get daemonset detail"},
		{Name: "DaemonSet 应用 YAML", Resource: "/api/v1/daemonsets/apply", Action: "POST", Description: "Apply daemonset yaml"},
		{Name: "DaemonSet 重启", Resource: "/api/v1/daemonsets/restart", Action: "POST", Description: "Restart daemonset"},
		{Name: "DaemonSet 关联 Pods", Resource: "/api/v1/daemonsets/pods", Action: "GET", Description: "List daemonset related pods"},
		{Name: "删除 DaemonSet", Resource: "/api/v1/daemonsets", Action: "DELETE", Description: "Delete daemonset"},

		{Name: "Job 列表", Resource: "/api/v1/jobs", Action: "GET", Description: "List jobs"},
		{Name: "Job 详情", Resource: "/api/v1/jobs/detail", Action: "GET", Description: "Get job detail"},
		{Name: "Job 关联 Pods", Resource: "/api/v1/jobs/pods", Action: "GET", Description: "List job related pods"},
		{Name: "Job 重新执行", Resource: "/api/v1/jobs/rerun", Action: "POST", Description: "Rerun a job"},
		{Name: "Job 应用 YAML", Resource: "/api/v1/jobs/apply", Action: "POST", Description: "Apply job yaml"},
		{Name: "删除 Job", Resource: "/api/v1/jobs", Action: "DELETE", Description: "Delete job"},

		{Name: "CronJob 列表", Resource: "/api/v1/cronjobs", Action: "GET", Description: "List cronjobs"},
		{Name: "CronJob 列表V2", Resource: "/api/v1/cronjobs/v2", Action: "GET", Description: "List cronjobs with suspend and last schedule"},
		{Name: "CronJob 详情", Resource: "/api/v1/cronjobs/detail", Action: "GET", Description: "Get cronjob detail"},
		{Name: "CronJob 关联 Pods", Resource: "/api/v1/cronjobs/pods", Action: "GET", Description: "List cronjob related pods"},
		{Name: "CronJob 应用 YAML", Resource: "/api/v1/cronjobs/apply", Action: "POST", Description: "Apply cronjob yaml"},
		{Name: "CronJob 暂停/恢复", Resource: "/api/v1/cronjobs/suspend", Action: "POST", Description: "Suspend/resume cronjob"},
		{Name: "CronJob 触发执行", Resource: "/api/v1/cronjobs/trigger", Action: "POST", Description: "Trigger cronjob once"},
		{Name: "删除 CronJob", Resource: "/api/v1/cronjobs", Action: "DELETE", Description: "Delete cronjob"},

		{Name: "ConfigMap 列表", Resource: "/api/v1/configmaps", Action: "GET", Description: "List configmaps"},
		{Name: "ConfigMap 详情", Resource: "/api/v1/configmaps/detail", Action: "GET", Description: "Get configmap detail"},
		{Name: "ConfigMap 应用 YAML", Resource: "/api/v1/configmaps/apply", Action: "POST", Description: "Apply configmap yaml"},
		{Name: "删除 ConfigMap", Resource: "/api/v1/configmaps", Action: "DELETE", Description: "Delete configmap"},

		{Name: "Secret 列表", Resource: "/api/v1/secrets", Action: "GET", Description: "List secrets"},
		{Name: "Secret 详情", Resource: "/api/v1/secrets/detail", Action: "GET", Description: "Get secret detail"},
		{Name: "Secret 应用 YAML", Resource: "/api/v1/secrets/apply", Action: "POST", Description: "Apply secret yaml"},
		{Name: "删除 Secret", Resource: "/api/v1/secrets", Action: "DELETE", Description: "Delete secret"},

		{Name: "Service 列表", Resource: "/api/v1/k8s-services", Action: "GET", Description: "List services"},
		{Name: "Service 详情", Resource: "/api/v1/k8s-services/detail", Action: "GET", Description: "Get service detail"},
		{Name: "Service 应用 YAML", Resource: "/api/v1/k8s-services/apply", Action: "POST", Description: "Apply service yaml"},
		{Name: "删除 Service", Resource: "/api/v1/k8s-services", Action: "DELETE", Description: "Delete service"},

		{Name: "PersistentVolume 列表", Resource: "/api/v1/persistentvolumes", Action: "GET", Description: "List persistent volumes"},
		{Name: "PersistentVolume 详情", Resource: "/api/v1/persistentvolumes/detail", Action: "GET", Description: "Get persistent volume detail"},
		{Name: "PersistentVolume 应用 YAML", Resource: "/api/v1/persistentvolumes/apply", Action: "POST", Description: "Apply persistent volume yaml"},
		{Name: "删除 PersistentVolume", Resource: "/api/v1/persistentvolumes", Action: "DELETE", Description: "Delete persistent volume"},

		{Name: "PersistentVolumeClaim 列表", Resource: "/api/v1/persistentvolumeclaims", Action: "GET", Description: "List persistent volume claims"},
		{Name: "PersistentVolumeClaim 详情", Resource: "/api/v1/persistentvolumeclaims/detail", Action: "GET", Description: "Get persistent volume claim detail"},
		{Name: "PersistentVolumeClaim 应用 YAML", Resource: "/api/v1/persistentvolumeclaims/apply", Action: "POST", Description: "Apply persistent volume claim yaml"},
		{Name: "删除 PersistentVolumeClaim", Resource: "/api/v1/persistentvolumeclaims", Action: "DELETE", Description: "Delete persistent volume claim"},

		{Name: "StorageClass 列表", Resource: "/api/v1/storageclasses", Action: "GET", Description: "List storage classes"},
		{Name: "StorageClass 详情", Resource: "/api/v1/storageclasses/detail", Action: "GET", Description: "Get storage class detail"},
		{Name: "StorageClass 应用 YAML", Resource: "/api/v1/storageclasses/apply", Action: "POST", Description: "Apply storage class yaml"},
		{Name: "删除 StorageClass", Resource: "/api/v1/storageclasses", Action: "DELETE", Description: "Delete storage class"},

		{Name: "Ingress 列表", Resource: "/api/v1/ingresses", Action: "GET", Description: "List ingresses"},
		{Name: "Ingress 详情", Resource: "/api/v1/ingresses/detail", Action: "GET", Description: "Get ingress detail"},
		{Name: "Ingress 应用 YAML", Resource: "/api/v1/ingresses/apply", Action: "POST", Description: "Apply ingress yaml"},
		{Name: "IngressClass 列表", Resource: "/api/v1/ingresses/classes", Action: "GET", Description: "List ingress classes"},
		{Name: "IngressClass 详情", Resource: "/api/v1/ingresses/classes/detail", Action: "GET", Description: "Get ingress class detail"},
		{Name: "IngressClass 应用 YAML", Resource: "/api/v1/ingresses/classes/apply", Action: "POST", Description: "Apply ingress class yaml"},
		{Name: "删除 IngressClass", Resource: "/api/v1/ingresses/classes", Action: "DELETE", Description: "Delete ingress class"},
		{Name: "重启 Ingress-Nginx Pods", Resource: "/api/v1/ingresses/nginx/restart", Action: "POST", Description: "Restart ingress-nginx controller pods to refresh cert"},
		{Name: "删除 Ingress", Resource: "/api/v1/ingresses", Action: "DELETE", Description: "Delete ingress"},
		{Name: "网络策略列表", Resource: "/api/v1/network-policies", Action: "GET", Description: "List network policies"},
		{Name: "网络策略详情", Resource: "/api/v1/network-policies/detail", Action: "GET", Description: "Get network policy detail"},
		{Name: "网络策略应用 YAML", Resource: "/api/v1/network-policies/apply", Action: "POST", Description: "Apply network policy yaml"},
		{Name: "删除网络策略", Resource: "/api/v1/network-policies", Action: "DELETE", Description: "Delete network policy"},

		{Name: "项目列表", Resource: "/api/v1/projects", Action: "GET", Description: "List projects"},
		{Name: "创建项目", Resource: "/api/v1/projects", Action: "POST", Description: "Create project"},
		{Name: "更新项目", Resource: "/api/v1/projects/:id", Action: "PUT", Description: "Update project"},
		{Name: "删除项目", Resource: "/api/v1/projects/:id", Action: "DELETE", Description: "Delete project"},
		{Name: "项目成员列表", Resource: "/api/v1/projects/:id/members", Action: "GET", Description: "List project members"},
		{Name: "添加项目成员", Resource: "/api/v1/projects/:id/members", Action: "POST", Description: "Add project member"},
		{Name: "更新项目成员", Resource: "/api/v1/projects/:id/members/:memberId", Action: "PUT", Description: "Update project member"},
		{Name: "移除项目成员", Resource: "/api/v1/projects/:id/members/:memberId", Action: "DELETE", Description: "Remove project member"},
		{Name: "项目服务器列表", Resource: "/api/v1/projects/:id/servers", Action: "GET", Description: "List project servers"},
		{Name: "项目服务器保存", Resource: "/api/v1/projects/:id/servers", Action: "POST", Description: "Upsert project server"},
		{Name: "项目服务器详情", Resource: "/api/v1/projects/:id/servers/:serverId", Action: "GET", Description: "Project server detail"},
		{Name: "删除项目服务器", Resource: "/api/v1/projects/:id/servers/:serverId", Action: "DELETE", Description: "Delete project server"},
		{Name: "项目服务器命令", Resource: "/api/v1/projects/:id/servers/:serverId/exec", Action: "POST", Description: "Exec on project server"},
		{Name: "项目服务器分组树", Resource: "/api/v1/projects/:id/server-groups/tree", Action: "GET", Description: "List server groups tree"},
		{Name: "项目服务器分组创建", Resource: "/api/v1/projects/:id/server-groups", Action: "POST", Description: "Upsert server group"},
		{Name: "项目服务器分组更新", Resource: "/api/v1/projects/:id/server-groups/:groupId", Action: "PUT", Description: "Update server group"},
		{Name: "项目服务器分组删除", Resource: "/api/v1/projects/:id/server-groups/:groupId", Action: "DELETE", Description: "Delete server group"},
		{Name: "项目云账号列表", Resource: "/api/v1/projects/:id/cloud-accounts", Action: "GET", Description: "List cloud accounts"},
		{Name: "项目云账号保存", Resource: "/api/v1/projects/:id/cloud-accounts", Action: "POST", Description: "Upsert cloud account"},
		{Name: "项目云账号更新", Resource: "/api/v1/projects/:id/cloud-accounts/:accountId", Action: "PUT", Description: "Update cloud account"},
		{Name: "项目云账号同步", Resource: "/api/v1/projects/:id/cloud-accounts/:accountId/sync", Action: "PUT", Description: "Sync cloud account"},
		{Name: "项目服务器导入", Resource: "/api/v1/projects/:id/servers/import", Action: "POST", Description: "Import servers"},
		{Name: "项目服务器导入模板", Resource: "/api/v1/projects/:id/servers/import-template", Action: "GET", Description: "Servers import template"},
		{Name: "项目服务器导出", Resource: "/api/v1/projects/:id/servers/export", Action: "GET", Description: "Export servers"},
		{Name: "项目服务器连通测试", Resource: "/api/v1/projects/:id/servers/test", Action: "POST", Description: "Test server connection"},
		{Name: "项目服务器批量测试", Resource: "/api/v1/projects/:id/servers/test/batch", Action: "POST", Description: "Batch test servers"},
		{Name: "项目服务器同步", Resource: "/api/v1/projects/:id/servers/sync", Action: "POST", Description: "Sync servers"},
		{Name: "项目服务列表", Resource: "/api/v1/projects/:id/services", Action: "GET", Description: "List project services"},
		{Name: "项目服务保存", Resource: "/api/v1/projects/:id/services", Action: "POST", Description: "Upsert project service"},
		{Name: "删除项目服务", Resource: "/api/v1/projects/:id/services/:serviceId", Action: "DELETE", Description: "Delete project service"},
		{Name: "项目日志源列表", Resource: "/api/v1/projects/:id/log-sources", Action: "GET", Description: "List log sources"},
		{Name: "项目日志源保存", Resource: "/api/v1/projects/:id/log-sources", Action: "POST", Description: "Upsert log source"},
		{Name: "删除项目日志源", Resource: "/api/v1/projects/:id/log-sources/:logSourceId", Action: "DELETE", Description: "Delete log source"},
		{Name: "项目 Agent 列表", Resource: "/api/v1/projects/:id/agents/list", Action: "GET", Description: "List agents"},
		{Name: "项目 Agent 心跳刷新", Resource: "/api/v1/projects/:id/agents/heartbeat-refresh", Action: "POST", Description: "Batch refresh agent heartbeat"},
		{Name: "项目 Agent 状态", Resource: "/api/v1/projects/:id/agents/status", Action: "GET", Description: "Agent status"},
		{Name: "项目 Agent 引导", Resource: "/api/v1/projects/:id/agents/bootstrap", Action: "POST", Description: "Agent bootstrap"},
		{Name: "项目 Agent 轮换令牌", Resource: "/api/v1/projects/:id/agents/rotate-token", Action: "POST", Description: "Rotate agent token"},
		{Name: "项目 Agent 发现列表", Resource: "/api/v1/projects/:id/agents/discovery", Action: "GET", Description: "Agent discovery list"},
		{Name: "项目日志流", Resource: "/api/v1/projects/:id/logs/stream", Action: "GET", Description: "Stream project logs"},
		{Name: "项目日志导出", Resource: "/api/v1/projects/:id/logs/export", Action: "GET", Description: "Export project logs"},
		{Name: "项目日志文件列表", Resource: "/api/v1/projects/:id/log-files", Action: "GET", Description: "List log files"},
		{Name: "项目日志单元", Resource: "/api/v1/projects/:id/log-units", Action: "GET", Description: "List log units"},
		{Name: "项目服务器终端 WS", Resource: "/api/v1/projects/:id/servers/:serverId/terminal/ws", Action: "GET", Description: "Server terminal websocket"},

		{Name: "Event 列表", Resource: "/api/v1/events", Action: "GET", Description: "List events"},
		{Name: "CRD 列表", Resource: "/api/v1/crds", Action: "GET", Description: "List custom resource definitions"},
		{Name: "CRD 详情", Resource: "/api/v1/crds/detail", Action: "GET", Description: "Get custom resource definition detail"},
		{Name: "CRD 应用 YAML", Resource: "/api/v1/crds/apply", Action: "POST", Description: "Apply custom resource definition yaml"},
		{Name: "删除 CRD", Resource: "/api/v1/crds", Action: "DELETE", Description: "Delete custom resource definition"},
		{Name: "CR 资源类型列表", Resource: "/api/v1/crs/resources", Action: "GET", Description: "List custom resource types"},
		{Name: "CR 实例列表", Resource: "/api/v1/crs", Action: "GET", Description: "List custom resources"},
		{Name: "CR 实例详情", Resource: "/api/v1/crs/detail", Action: "GET", Description: "Get custom resource detail"},
		{Name: "CR 实例应用 YAML", Resource: "/api/v1/crs/apply", Action: "POST", Description: "Apply custom resource yaml"},
		{Name: "删除 CR 实例", Resource: "/api/v1/crs", Action: "DELETE", Description: "Delete custom resource"},
	}
}

func seedMenus(ctx context.Context, db *gorm.DB) error {
	if err := cleanupDuplicateKubernetesMenu(ctx, db); err != nil {
		return err
	}

	menus := defaultMenus()
	for i := range menus {
		if err := upsertMenu(ctx, db, &menus[i], 0); err != nil {
			return err
		}
		for j := range menus[i].Children {
			if err := upsertMenu(ctx, db, &menus[i].Children[j], menus[i].ID); err != nil {
				return err
			}
		}
	}
	// 再清理一次：防止历史脏数据与 upsert 互相叠加导致仍有重复目录
	return cleanupDuplicateKubernetesMenu(ctx, db)
}

func cleanupDuplicateKubernetesMenu(ctx context.Context, db *gorm.DB) error {
	var roots []model.Menu
	if err := db.WithContext(ctx).
		Where("TRIM(path) IN ?", []string{"/kubernetes", "/kubernetes/"}).
		Find(&roots).Error; err != nil {
		return err
	}
	if len(roots) <= 1 {
		return nil
	}

	keepID := roots[0].ID
	keepScore := -1
	for _, r := range roots {
		var children []model.Menu
		if err := db.WithContext(ctx).
			Where("parent_id = ?", r.ID).
			Find(&children).Error; err != nil {
			return err
		}
		score := 0
		for _, c := range children {
			switch c.Path {
			case "/pods", "/clusters", "/cronjobs", "/jobs", "/events":
				score++
			case "/pod", "/cluster":
				score--
			}
		}
		if score > keepScore {
			keepScore = score
			keepID = r.ID
		}
	}

	for _, r := range roots {
		if r.ID == keepID {
			continue
		}
		ids, err := collectMenuSubtreeIDs(ctx, db, r.ID)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			continue
		}
		if err := db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
	}

	return nil
}

func collectMenuSubtreeIDs(ctx context.Context, db *gorm.DB, rootID uint) ([]uint, error) {
	var out []uint
	queue := []uint{rootID}
	seen := map[uint]bool{}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)

		var children []model.Menu
		if err := db.WithContext(ctx).
			Where("parent_id = ?", id).
			Find(&children).Error; err != nil {
			return nil, err
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return out, nil
}

func upsertMenu(ctx context.Context, db *gorm.DB, menu *model.Menu, parentID uint) error {
	var existing model.Menu
	query := db.WithContext(ctx).Where("name = ?", menu.Name)
	if parentID == 0 {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}
	err := query.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		menu.ParentID = nil
		if parentID > 0 {
			p := parentID
			menu.ParentID = &p
		}
		return db.WithContext(ctx).Create(menu).Error
	}
	if err != nil {
		return err
	}

	existing.Path = menu.Path
	existing.Icon = menu.Icon
	existing.Sort = menu.Sort
	existing.Hidden = menu.Hidden
	existing.AdminOnly = menu.AdminOnly
	existing.Component = menu.Component
	existing.Redirect = menu.Redirect
	existing.Status = menu.Status
	// 以前如果已有同名菜单但 parent_id 写错，重新 seed 时旧逻辑不会修正 parent_id。
	// 这里显式把 parent_id 修正到目标结构，保证菜单树能正确挂到“系统管理”下面。
	existing.ParentID = nil
	if parentID > 0 {
		p := parentID
		existing.ParentID = &p
	}
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*menu = existing
	return nil
}

func defaultMenus() []model.Menu {
	return []model.Menu{
		{
			Name:      "资产总览",
			Path:      "/",
			Icon:      "DatabaseOutlined",
			Sort:      1,
			Component: "",
			Status:    1,
		},
		{
			Name:   "告警通知",
			Path:   "/alert-notify",
			Icon:   "BellOutlined",
			Sort:   2,
			Status: 1,
			Children: []model.Menu{
				{Name: "Webhook 告警通道", Path: "/alert-channels", Icon: "NotificationOutlined", Sort: 1, Component: "alert-channels-page", Status: 1},
				{Name: "告警监控平台", Path: "/alert-monitor-platform", Icon: "MonitorOutlined", Sort: 2, Component: "alert-monitor-platform-page", Status: 1},
			},
		},
		{
			Name:   "项目管理",
			Path:   "/project-management",
			Icon:   "ProjectOutlined",
			Sort:   4,
			Status: 1,
			Children: []model.Menu{
				{Name: "项目列表", Path: "/projects", Icon: "AppstoreOutlined", Sort: 1, Component: "projects-page", Status: 1},
				{Name: "项目成员", Path: "/project-members", Icon: "TeamOutlined", Sort: 2, Component: "project-members-page", Status: 1},
				{Name: "服务器管理", Path: "/project-servers", Icon: "HddOutlined", Sort: 3, Component: "project-servers-page", Status: 1},
				{Name: "服务配置", Path: "/project-services", Icon: "SettingOutlined", Sort: 4, Component: "project-services-page", Status: 1},
				{Name: "日志源配置", Path: "/project-log-sources", Icon: "FileSearchOutlined", Sort: 5, Component: "project-log-sources-page", Status: 1},
				{Name: "日志平台", Path: "/project-logs", Icon: "FileTextOutlined", Sort: 6, Component: "project-logs-page", Status: 1},
				{Name: "Agent 列表", Path: "/agent-list", Icon: "RobotOutlined", Sort: 7, Component: "agent-list-page", Status: 1},
			},
		},
		{
			Name:   "系统管理",
			Path:   "/system",
			Icon:   "SettingOutlined",
			Sort:   4,
			Status: 1,
			Children: []model.Menu{
				{Name: "账号管理", Path: "/users", Icon: "TeamOutlined", Sort: 1, Component: "", Status: 1},
				{Name: "角色管理", Path: "/roles", Icon: "ApartmentOutlined", Sort: 2, Component: "", Status: 1},
				{Name: "API管理", Path: "/permissions", Icon: "ApiOutlined", Sort: 3, Component: "", Status: 1},
				{Name: "授权管理", Path: "/policies", Icon: "AuditOutlined", Sort: 4, Component: "", Status: 1},
				{Name: "K8s 三元策略", Path: "/k8s-scoped-policies", Icon: "AuditOutlined", Sort: 5, Component: "k8s-scoped-policies-page", Status: 1},
				{Name: "注册审核", Path: "/registrations", Icon: "CheckCircleOutlined", Sort: 5, Component: "", Status: 1},
				{Name: "菜单管理", Path: "/menus", Icon: "MenuOutlined", Sort: 6, Component: "", Status: 1},
				{Name: "登录日志", Path: "/login-logs", Icon: "LoginOutlined", Sort: 8, Component: "", Status: 1},
				{Name: "操作历史", Path: "/operation-logs", Icon: "HistoryOutlined", Sort: 9, Component: "", Status: 1},
				{Name: "封禁 IP 管理", Path: "/banned-ips", Icon: "ApiOutlined", Sort: 10, Component: "", Status: 1, AdminOnly: false},
			},
		},
		{
			Name:   "Kubernetes 容器管理",
			Path:   "/kubernetes",
			Icon:   "KubernetesOutlined",
			Sort:   5,
			Status: 1,
			Children: []model.Menu{
				{Name: "集群管理", Path: "/clusters", Icon: "KubernetesOutlined", Sort: 1, Component: "", Status: 1},
				{Name: "命名空间管理", Path: "/namespaces", Icon: "AppstoreOutlined", Sort: 2, Component: "namespaces-page", Status: 1},
				{Name: "Node 管理", Path: "/nodes", Icon: "HddOutlined", Sort: 3, Component: "nodes-page", Status: 1},
				{Name: "组件状态", Path: "/component-status", Icon: "HeartOutlined", Sort: 4, Component: "component-status-page", Status: 1},
				{Name: "Pod 管理", Path: "/pods", Icon: "KubernetesOutlined", Sort: 5, Component: "", Status: 1},
				{Name: "Deployment 管理", Path: "/deployments", Icon: "DeploymentUnitOutlined", Sort: 6, Component: "deployments-page", Status: 1},
				{Name: "StatefulSet 管理", Path: "/statefulsets", Icon: "ClusterOutlined", Sort: 7, Component: "statefulsets-page", Status: 1},
				{Name: "DaemonSet 管理", Path: "/daemonsets", Icon: "ApiOutlined", Sort: 8, Component: "daemonsets-page", Status: 1},
				{Name: "CronJob 管理", Path: "/cronjobs", Icon: "ClockCircleOutlined", Sort: 9, Component: "cronjobs-page", Status: 1},
				{Name: "Job 管理", Path: "/jobs", Icon: "ThunderboltOutlined", Sort: 10, Component: "jobs-page", Status: 1},
				{Name: "ConfigMap 管理", Path: "/configmaps", Icon: "FileTextOutlined", Sort: 11, Component: "configmaps-page", Status: 1},
				{Name: "Secret 管理", Path: "/secrets", Icon: "SafetyOutlined", Sort: 12, Component: "secrets-page", Status: 1},
				{Name: "Service 管理", Path: "/k8s-services", Icon: "ApartmentOutlined", Sort: 13, Component: "k8s-services-page", Status: 1},
				{Name: "PersistentVolume", Path: "/persistentvolumes", Icon: "DatabaseOutlined", Sort: 14, Component: "persistentvolumes-page", Status: 1},
				{Name: "PersistentVolumeClaim", Path: "/persistentvolumeclaims", Icon: "HddOutlined", Sort: 15, Component: "persistentvolumeclaims-page", Status: 1},
				{Name: "StorageClass", Path: "/storageclasses", Icon: "FolderOpenOutlined", Sort: 16, Component: "storageclasses-page", Status: 1},
				{Name: "Ingress 管理", Path: "/ingresses", Icon: "GatewayOutlined", Sort: 17, Component: "ingresses-page", Status: 1},
				{Name: "IngressClass 入口类", Path: "/ingress-classes", Icon: "GatewayOutlined", Sort: 18, Component: "ingress-classes-page", Status: 1},
				{Name: "网络策略管理", Path: "/network-policies", Icon: "DeploymentUnitOutlined", Sort: 19, Component: "network-policies-page", Status: 1},
				{Name: "Event 事件", Path: "/events", Icon: "FileSearchOutlined", Sort: 20, Component: "events-page", Status: 1},
			},
		},
		{
			Name:   "Kubernetes CRD 管理",
			Path:   "/kubernetes-crd",
			Icon:   "BranchesOutlined",
			Sort:   5,
			Status: 1,
			Children: []model.Menu{
				{Name: "CRD 管理", Path: "/crds", Icon: "BranchesOutlined", Sort: 1, Component: "crds-page", Status: 1},
				{Name: "CR 实例管理", Path: "/crs", Icon: "DatabaseOutlined", Sort: 2, Component: "crs-page", Status: 1},
			},
		},
	}
}
