package service

import (
	"sort"
	"strings"

	"yunshu/internal/model"
)

const (
	k8sScopeEnableTag  = "[k8s-scope=on]"
	k8sScopeDisableTag = "[k8s-scope=off]"
)

func K8sScopeRouteKey(path, method string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func BuildK8sScopeMappings(perms []model.Permission) (map[string]string, map[string]bool) {
	actionByRoute := make(map[string]string)
	scopedRoutes := make(map[string]bool)
	for _, p := range perms {
		path := strings.TrimSpace(p.Resource)
		method := strings.ToUpper(strings.TrimSpace(p.Action))
		if path == "" || method == "" {
			continue
		}
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		if !isScopedK8sPermission(p) {
			continue
		}
		code := autoActionCode(path, method)
		if code == "" {
			continue
		}
		key := K8sScopeRouteKey(path, method)
		actionByRoute[key] = code
		scopedRoutes[key] = true
	}
	return actionByRoute, scopedRoutes
}

func BuildK8sScopeActionCatalog(perms []model.Permission) ([]K8sActionItem, []string) {
	actionByCode := make(map[string]K8sActionItem)
	pathSet := make(map[string]bool)
	for _, p := range perms {
		path := strings.TrimSpace(p.Resource)
		method := strings.ToUpper(strings.TrimSpace(p.Action))
		if path == "" || method == "" {
			continue
		}
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		if !isScopedK8sPermission(p) {
			continue
		}
		code := autoActionCode(path, method)
		if code == "" {
			continue
		}
		pathSet[path] = true
		if _, ok := actionByCode[code]; ok {
			continue
		}
		actionByCode[code] = K8sActionItem{
			Code:        code,
			Name:        p.Name,
			Description: p.Description,
		}
	}

	actions := make([]K8sActionItem, 0, len(actionByCode))
	for _, item := range actionByCode {
		actions = append(actions, item)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Code < actions[j].Code })

	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return actions, paths
}

func isScopedK8sPermission(p model.Permission) bool {
	if p.K8sScopeEnabled {
		return true
	}
	path := strings.TrimSpace(p.Resource)
	method := strings.ToUpper(strings.TrimSpace(p.Action))
	desc := strings.ToLower(strings.TrimSpace(p.Description))
	if strings.Contains(desc, strings.ToLower(k8sScopeDisableTag)) {
		return false
	}
	if strings.Contains(desc, strings.ToLower(k8sScopeEnableTag)) {
		return true
	}

	// 默认策略：高风险/变更类接口纳入三元授权，读接口默认不纳入。
	m := strings.ToUpper(strings.TrimSpace(method))
	if strings.Contains(path, "/exec") {
		return true
	}
	return m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE"
}

func autoActionCode(path, method string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(path), "/api/v1/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return ""
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, p := range rawParts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" || strings.HasPrefix(p, ":") {
			continue
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return ""
	}

	verbHints := map[string]bool{
		"apply":   true,
		"delete":  true,
		"scale":   true,
		"restart": true,
		"suspend": true,
		"trigger": true,
		"rerun":   true,
		"exec":    true,
		"upload":  true,
		"update":  true,
		"create":  true,
	}

	op := ""
	if tail := parts[len(parts)-1]; verbHints[tail] {
		op = tail
		parts = parts[:len(parts)-1]
	}
	if op == "" {
		switch strings.ToUpper(strings.TrimSpace(method)) {
		case "POST":
			op = "create"
		case "PUT":
			op = "update"
		case "PATCH":
			op = "patch"
		case "DELETE":
			op = "delete"
		default:
			op = "get"
		}
	}

	resource := parts[0]
	if len(parts) >= 2 {
		if resource == "ingresses" && parts[1] == "classes" {
			resource = "ingressclasses"
		} else if resource == "rbac" {
			resource = "rbac-" + parts[1]
		}
	}
	return resource + "/" + op
}
