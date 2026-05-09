package k8sutil

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// PatchResourceList 将 patch 合并进 ResourceList：空字符串表示删除该资源键；nil patch 不修改。
func PatchResourceList(list *corev1.ResourceList, patch map[string]string) error {
	if patch == nil {
		return nil
	}
	for k, v := range patch {
		rn := corev1.ResourceName(strings.TrimSpace(k))
		if rn == "" {
			continue
		}
		val := strings.TrimSpace(v)
		if val == "" {
			if *list != nil {
				delete(*list, rn)
			}
			continue
		}
		q, err := resource.ParseQuantity(val)
		if err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
		if *list == nil {
			*list = corev1.ResourceList{}
		}
		(*list)[rn] = q
	}
	return nil
}
