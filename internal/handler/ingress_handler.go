package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type IngressHandler struct {
	svc *service.K8sIngressService
}

func NewIngressHandler(svc *service.K8sIngressService) *IngressHandler {
	return &IngressHandler{svc: svc}
}

func (h *IngressHandler) List(c *gin.Context) {
	var q service.IngressListQuery
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

func (h *IngressHandler) Detail(c *gin.Context) {
	var q service.IngressDetailQuery
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

func (h *IngressHandler) Apply(c *gin.Context) {
	var req service.IngressApplyRequest
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

func (h *IngressHandler) Delete(c *gin.Context) {
	var req service.IngressDeleteRequest
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

func (h *IngressHandler) RestartNginxPods(c *gin.Context) {
	var req service.IngressNginxRestartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.RestartIngressNginxPods(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *IngressHandler) ListClasses(c *gin.Context) {
	var q service.IngressClassListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListClasses(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *IngressHandler) DetailClass(c *gin.Context) {
	var q service.IngressClassDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.DetailClass(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *IngressHandler) ApplyClass(c *gin.Context) {
	var req service.IngressClassApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.ApplyClass(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *IngressHandler) DeleteClass(c *gin.Context) {
	var req service.IngressClassDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteClass(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}
