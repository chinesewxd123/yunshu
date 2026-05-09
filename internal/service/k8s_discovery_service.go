package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/constants"

	"k8s.io/client-go/kubernetes"
)

// APIResourceDiscoveryItem 单条 API 资源元数据（对齐 kubectl api-resources -o wide 的常见字段）。
type APIResourceDiscoveryItem struct {
	GroupVersion string   `json:"group_version"`
	Name         string   `json:"name"`
	Namespaced   bool     `json:"namespaced"`
	Kind         string   `json:"kind"`
	Verbs        []string `json:"verbs"`
}

type K8sDiscoveryService struct {
	runtime *K8sRuntimeService
}

func NewK8sDiscoveryService(runtime *K8sRuntimeService) *K8sDiscoveryService {
	return &K8sDiscoveryService{runtime: runtime}
}

// ListAPIResources 返回集群支持的 API 资源列表（排除 status/scale 等子资源名中带 / 的条目）。
func (s *K8sDiscoveryService) ListAPIResources(ctx context.Context, clusterID uint, namespaced *bool) ([]APIResourceDiscoveryItem, error) {
	_, cfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, constants.ErrInternalWithMsg(fmt.Sprintf(constants.ErrFmt6d3ec85d0a18, err))
	}

	_, lists, err := cs.Discovery().ServerGroupsAndResources()
	if err != nil {
		if lists == nil || len(lists) == 0 {
			return nil, constants.ErrInternalWithMsg(fmt.Sprintf("discovery: %v", err))
		}
	}

	out := make([]APIResourceDiscoveryItem, 0, 256)
	for _, rl := range lists {
		gv := strings.TrimSpace(rl.GroupVersion)
		for _, r := range rl.APIResources {
			name := strings.TrimSpace(r.Name)
			if name == "" || strings.Contains(name, "/") {
				continue
			}
			if namespaced != nil && r.Namespaced != *namespaced {
				continue
			}
			verbs := append([]string(nil), r.Verbs...)
			sort.Strings(verbs)
			out = append(out, APIResourceDiscoveryItem{
				GroupVersion: gv,
				Name:         name,
				Namespaced:   r.Namespaced,
				Kind:         r.Kind,
				Verbs:        verbs,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupVersion != out[j].GroupVersion {
			return out[i].GroupVersion < out[j].GroupVersion
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
