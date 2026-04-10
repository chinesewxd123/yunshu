package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NamespaceHandler struct {
	svc *service.K8sNamespaceService
}

func NewNamespaceHandler(svc *service.K8sNamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

func (h *NamespaceHandler) List(c *gin.Context) {
	var query service.NamespaceListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *NamespaceHandler) Detail(c *gin.Context) {
	var query service.NamespaceDetailQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *NamespaceHandler) Apply(c *gin.Context) {
	var req service.NamespaceApplyRequest
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

func (h *NamespaceHandler) Delete(c *gin.Context) {
	var req service.NamespaceDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}
