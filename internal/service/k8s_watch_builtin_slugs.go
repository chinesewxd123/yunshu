package service

import (
	"fmt"
	"strings"
	"yunshu/internal/pkg/constants"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// builtinWatchSlugMap：Watch SSE 短名模式与 GVR 映射（与动态 GVR+RESTMapper 模式互补；单表维护避免散落魔法字串）。
var builtinWatchSlugMap = map[string]watchGVR{
	"pods":                 {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, Namespaced: true},
	"events":               {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}, Namespaced: true},
	"configmaps":           {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, Namespaced: true},
	"secrets":              {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}, Namespaced: true},
	"services":             {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, Namespaced: true},
	"persistentvolumeclaims": {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}, Namespaced: true},
	"namespaces":           {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, Namespaced: false},
	"nodes":                {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, Namespaced: false},
	"persistentvolumes":    {GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}, Namespaced: false},
	"deployments":          {GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, Namespaced: true},
	"statefulsets":         {GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, Namespaced: true},
	"daemonsets":           {GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, Namespaced: true},
	"replicasets":          {GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, Namespaced: true},
	"jobs":                 {GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}, Namespaced: true},
	"cronjobs":             {GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, Namespaced: true},
	"ingresses":            {GVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, Namespaced: true},
	"networkpolicies":      {GVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, Namespaced: true},
	"horizontalpodautoscalers": {
		GVR: schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, Namespaced: true,
	},
}

func builtinWatchGVRBySlug(slug string) (watchGVR, error) {
	key := strings.ToLower(strings.TrimSpace(slug))
	def, ok := builtinWatchSlugMap[key]
	if !ok {
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("不支持的 resource=%s", slug))
	}
	return def, nil
}
