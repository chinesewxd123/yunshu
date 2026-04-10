package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ServiceResourceHandler struct {
	svc *service.K8sServiceResourceService
}

func NewServiceResourceHandler(svc *service.K8sServiceResourceService) *ServiceResourceHandler {
	return &ServiceResourceHandler{svc: svc}
}

func (h *ServiceResourceHandler) List(c *gin.Context) {
	var q service.K8sServiceListQuery
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

func (h *ServiceResourceHandler) Detail(c *gin.Context) {
	var q service.K8sServiceDetailQuery
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

func (h *ServiceResourceHandler) Apply(c *gin.Context) {
	var req service.K8sServiceApplyRequest
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

func (h *ServiceResourceHandler) Delete(c *gin.Context) {
	var req service.K8sServiceDeleteRequest
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
