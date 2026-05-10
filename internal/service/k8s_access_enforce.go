package service

import (
	"strings"

	"yunshu/internal/model"
)

// K8sAccessTierRank 与 repository 中档位序一致，用于比较。
const (
	K8sAccessRankNone         = 0
	K8sAccessRankReadonly     = 1
	K8sAccessRankReadonlyExec = 2
	K8sAccessRankAdmin        = 3
)

// RequiredK8sAccessRank 根据权限目录展开结果，计算访问某路由所需的最低档位序。
func RequiredK8sAccessRank(perms []model.Permission, routePath, httpMethod, actionCode string) int {
	path := strings.TrimSpace(routePath)
	method := strings.ToUpper(strings.TrimSpace(httpMethod))
	code := strings.TrimSpace(actionCode)
	key := path + "\x00" + code

	readonly := expandReadonly(perms)
	reExec := expandPresetTriples(perms, PresetK8sReadonlyExec)
	admin := expandPresetTriples(perms, PresetK8sAdmin)

	if containsPolicyPathKey(readonly, key) {
		return K8sAccessRankReadonly
	}
	if containsPolicyPathKey(reExec, key) {
		return K8sAccessRankReadonlyExec
	}
	if containsPolicyPathKey(admin, key) {
		return K8sAccessRankAdmin
	}
	// 兜底：Exec / 终端先于「GET + k8s 只读前缀」，避免 /pods/exec/ws 被误判为只读档
	if strings.Contains(strings.ToLower(path), "exec") || strings.Contains(strings.ToLower(code), "exec") {
		return K8sAccessRankReadonlyExec
	}
	if method == "GET" && IsK8sReadAPIPath(path) {
		return K8sAccessRankReadonly
	}
	return K8sAccessRankAdmin
}

func containsPolicyPathKey(list []policyPathAction, key string) bool {
	for _, x := range list {
		if x.path+"\x00"+x.action == key {
			return true
		}
	}
	return false
}
