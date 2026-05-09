package service

import (
	"strings"

	"yunshu/internal/model"
)

// K8sClusterAccessPreset 对齐 k8m 的集群侧能力档位：只读、只读+Exec、集群管理（变更类全含）。
type K8sClusterAccessPreset string

const (
	PresetK8sReadonly     K8sClusterAccessPreset = "readonly"
	PresetK8sReadonlyExec K8sClusterAccessPreset = "readonly_exec"
	PresetK8sAdmin        K8sClusterAccessPreset = "admin"
)

type policyPathAction struct {
	path   string
	action string
}

func expandPresetTriples(perms []model.Permission, preset K8sClusterAccessPreset) []policyPathAction {
	switch preset {
	case PresetK8sReadonly:
		return expandReadonly(perms)
	case PresetK8sReadonlyExec:
		return unionTriples(expandReadonly(perms), expandExecExtras(perms))
	case PresetK8sAdmin:
		return expandAdmin(perms)
	default:
		return nil
	}
}

func expandReadonly(perms []model.Permission) []policyPathAction {
	var out []policyPathAction
	seen := make(map[string]struct{})
	for _, p := range perms {
		path := strings.TrimSpace(p.Resource)
		method := strings.ToUpper(strings.TrimSpace(p.Action))
		if path == "" || method != "GET" {
			continue
		}
		if !IsK8sReadAPIPath(path) {
			continue
		}
		code := autoActionCode(path, method)
		if code == "" {
			continue
		}
		key := path + "\x00" + code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, policyPathAction{path: path, action: code})
	}
	return out
}

func expandExecExtras(perms []model.Permission) []policyPathAction {
	var out []policyPathAction
	seen := make(map[string]struct{})
	for _, p := range perms {
		path := strings.TrimSpace(p.Resource)
		method := strings.ToUpper(strings.TrimSpace(p.Action))
		if path == "" || method == "" {
			continue
		}
		if !isScopedK8sPermission(p) {
			continue
		}
		code := autoActionCode(path, method)
		if code == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(path), "exec") && !strings.Contains(strings.ToLower(code), "exec") {
			continue
		}
		key := path + "\x00" + code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, policyPathAction{path: path, action: code})
	}
	return out
}

func expandAdmin(perms []model.Permission) []policyPathAction {
	var out []policyPathAction
	seen := make(map[string]struct{})
	for _, p := range perms {
		path := strings.TrimSpace(p.Resource)
		method := strings.ToUpper(strings.TrimSpace(p.Action))
		if path == "" || method == "" {
			continue
		}
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		include := isScopedK8sPermission(p)
		if !include && method == "GET" && IsK8sReadAPIPath(path) {
			include = true
		}
		if !include {
			continue
		}
		code := autoActionCode(path, method)
		if code == "" {
			continue
		}
		key := path + "\x00" + code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, policyPathAction{path: path, action: code})
	}
	return out
}

func unionTriples(a, b []policyPathAction) []policyPathAction {
	seen := make(map[string]struct{})
	var out []policyPathAction
	for _, x := range append(a, b...) {
		key := x.path + "\x00" + x.action
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, x)
	}
	return out
}
