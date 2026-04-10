package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	svc *service.K8sConfigService
}

func NewConfigHandler(svc *service.K8sConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// ConfigMaps
func (h *ConfigHandler) ListConfigMaps(c *gin.Context) {
	var q service.ConfigListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListConfigMaps(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ConfigHandler) ConfigMapDetail(c *gin.Context) {
	var q service.ConfigDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.ConfigMapDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ConfigHandler) DeleteConfigMap(c *gin.Context) {
	var req service.ConfigDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteConfigMap(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// Secrets
func (h *ConfigHandler) ListSecrets(c *gin.Context) {
	var q service.ConfigListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListSecrets(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ConfigHandler) SecretDetail(c *gin.Context) {
	var q service.ConfigDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.SecretDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ConfigHandler) DeleteSecret(c *gin.Context) {
	var req service.ConfigDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteSecret(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// shared apply
func (h *ConfigHandler) Apply(c *gin.Context) {
	var req service.ConfigApplyRequest
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
