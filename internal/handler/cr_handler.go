package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type CRHandler struct {
	svc *service.K8sCRService
}

func NewCRHandler(svc *service.K8sCRService) *CRHandler {
	return &CRHandler{svc: svc}
}

func (h *CRHandler) ListResources(c *gin.Context) {
	var q service.CRResourceListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListResources(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CRHandler) List(c *gin.Context) {
	var q service.CRListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CRHandler) Detail(c *gin.Context) {
	var q service.CRDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *CRHandler) Apply(c *gin.Context) {
	var req service.CRApplyRequest
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

func (h *CRHandler) Delete(c *gin.Context) {
	var req service.CRDeleteRequest
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
