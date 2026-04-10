package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type StorageHandler struct {
	svc *service.K8sStorageService
}

func NewStorageHandler(svc *service.K8sStorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

func (h *StorageHandler) ListPVs(c *gin.Context) {
	var q service.StorageListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListPVs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *StorageHandler) ListPVCs(c *gin.Context) {
	var q service.StorageListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListPVCs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *StorageHandler) ListStorageClasses(c *gin.Context) {
	var q service.StorageListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListStorageClasses(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *StorageHandler) Detail(c *gin.Context) {
	kind := c.Query("kind")
	var q service.StorageDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), kind, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *StorageHandler) Apply(c *gin.Context) {
	var req service.StorageApplyRequest
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

func (h *StorageHandler) Delete(c *gin.Context) {
	kind := c.Query("kind")
	var req service.StorageDeleteRequest
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
