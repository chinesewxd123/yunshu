package handler

import (
	"context"

	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type RBACHandler struct {
	svc *service.K8sRBACService
}

func NewRBACHandler(svc *service.K8sRBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListRoles(ctx, query)
		return gin.H{"list": list}, err
	})
}

func (h *RBACHandler) ListRoleBindings(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListRoleBindings(ctx, query)
		return gin.H{"list": list}, err
	})
}

func (h *RBACHandler) ListClusterRoles(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListClusterRoles(ctx, query)
		return gin.H{"list": list}, err
	})
}

func (h *RBACHandler) ListClusterRoleBindings(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.RbacListQuery) (gin.H, error) {
		list, err := h.svc.ListClusterRoleBindings(ctx, query)
		return gin.H{"list": list}, err
	})
}

func (h *RBACHandler) Detail(c *gin.Context) {
	handleQueryWithKind(c, h.svc.Detail)
}

func (h *RBACHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *RBACHandler) Delete(c *gin.Context) {
	handleQueryWithKindOK(c, true, h.svc.Delete)
}
