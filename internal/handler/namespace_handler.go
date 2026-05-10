package handler

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type NamespaceHandler struct {
	svc *service.K8sNamespaceService
}

// NewNamespaceHandler 创建相关逻辑。
func NewNamespaceHandler(svc *service.K8sNamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) List(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, query service.NamespaceListQuery) ([]service.NamespaceListItem, error) {
		user, ok := auth.CurrentUserFromContext(c)
		var pack *k8sauth.PrincipalPack
		if ok && user != nil {
			p := k8sauth.PackFromCurrentUser(user)
			pack = &p
		}
		return h.svc.List(ctx, query, pack)
	})
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Detail(c *gin.Context) {
	ServeQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Apply(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Delete(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.Delete)
}
