package constants

// K8sClusterPermissionPathPrefixes 与 internal/router/router.go 中挂载了 K8sScopeAuthorize 的路由组前缀一致。
// permissions.resource 以此任一前缀开头（含带 :id 的路径）即视为「集群资源接口」，供 API 管理页筛选。
// 后端新增/下线 K8s 范围中间件路由时请同步更新本列表。
var K8sClusterPermissionPathPrefixes = []string{
	"/api/v1/clusters",
	"/api/v1/pods",
	"/api/v1/namespaces",
	"/api/v1/nodes",
	"/api/v1/deployments",
	"/api/v1/statefulsets",
	"/api/v1/daemonsets",
	"/api/v1/cronjobs",
	"/api/v1/jobs",
	"/api/v1/configmaps",
	"/api/v1/secrets",
	"/api/v1/persistentvolumes",
	"/api/v1/persistentvolumeclaims",
	"/api/v1/storageclasses",
	"/api/v1/k8s-services",
	"/api/v1/ingresses",
	"/api/v1/network-policies",
	"/api/v1/horizontal-pod-autoscalers",
	"/api/v1/k8s/resource-watch",
	"/api/v1/events",
	"/api/v1/crds",
	"/api/v1/crs",
	"/api/v1/rbac",
	"/api/v1/serviceaccounts",
	"/api/v1/k8s/event-forward",
}
