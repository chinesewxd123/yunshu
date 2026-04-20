package handler

import (
	"context"

	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type RBACHandler struct {
	svc *service.K8sRBACService
}

// NewRBACHandler 创建相关逻辑。
func NewRBACHandler(svc *service.K8sRBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// ListRoles 查询列表对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) ListRoles(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListRoles(ctx, query)
		return gin.H{"list": list}, err
	})
}

// ListRoleBindings 查询列表对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) ListRoleBindings(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListRoleBindings(ctx, query)
		return gin.H{"list": list}, err
	})
}

// ListClusterRoles 查询列表对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) ListClusterRoles(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListClusterRoles(ctx, query)
		return gin.H{"list": list}, err
	})
}

// ListClusterRoleBindings 查询列表对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) ListClusterRoleBindings(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListClusterRoleBindings(ctx, query)
		return gin.H{"list": list}, err
	})
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) Detail(c *gin.Context) {
	handleQueryWithKind(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *RBACHandler) Delete(c *gin.Context) {
	handleQueryWithKindOK(c, true, h.svc.Delete)
}
