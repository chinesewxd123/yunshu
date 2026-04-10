package handler

import (
	"go-permission-system/internal/pkg/response"
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
	var query service.RbacListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListRoles(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *RBACHandler) ListRoleBindings(c *gin.Context) {
	var query service.RbacListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListRoleBindings(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *RBACHandler) ListClusterRoles(c *gin.Context) {
	var query service.RbacListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListClusterRoles(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *RBACHandler) ListClusterRoleBindings(c *gin.Context) {
	var query service.RbacListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListClusterRoleBindings(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *RBACHandler) Detail(c *gin.Context) {
	kind := c.Query("kind")
	var query service.RbacNameQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), kind, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *RBACHandler) Apply(c *gin.Context) {
	var req service.RbacApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Apply(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *RBACHandler) Delete(c *gin.Context) {
	kind := c.Query("kind")
	var req service.RbacDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), kind, req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}
