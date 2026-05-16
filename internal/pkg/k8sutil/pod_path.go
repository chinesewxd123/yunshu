package k8sutil

import (
	"path"
	"strings"

	"yunshu/internal/pkg/constants"
)

// ValidatePodContainerPath 校验容器内文件路径，禁止目录穿越。
func ValidatePodContainerPath(raw string) error {
	p := strings.TrimSpace(raw)
	if p == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg72b2ecec3b64)
	}
	if !strings.HasPrefix(p, "/") {
		return constants.ErrBadRequestWithMsg("容器路径必须以 / 开头")
	}
	if strings.Contains(p, "..") {
		return constants.ErrBadRequestWithMsg("路径不允许包含 ..")
	}
	clean := path.Clean(p)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return constants.ErrBadRequestWithMsg("路径不允许访问上级目录")
	}
	return nil
}
