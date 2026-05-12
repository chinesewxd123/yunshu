package service

import "strings"

// IsK8sReadAPIPath 判断是否为控制台使用的「资源列表/详情」类只读 API（与 Authorize 兜底 allowReadByK8sScopedPolicy 对齐）。
func IsK8sReadAPIPath(path string) bool {
	p := strings.TrimSpace(path)
	k8sPrefixes := []string{
		"/api/v1/k8s-policies/cluster-auth-matrix",
		"/api/v1/k8s-policies/user-cluster-auth",
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
		"/api/v1/k8s-services",
		"/api/v1/persistentvolumes",
		"/api/v1/persistentvolumeclaims",
		"/api/v1/storageclasses",
		"/api/v1/ingresses",
		"/api/v1/events",
		"/api/v1/crds",
		"/api/v1/crs",
		"/api/v1/rbac",
	}
	for _, prefix := range k8sPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
