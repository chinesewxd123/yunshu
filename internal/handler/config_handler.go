package handler

import (
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
	handleQuery(c, h.svc.ListConfigMaps)
}

func (h *ConfigHandler) ConfigMapDetail(c *gin.Context) {
	handleQuery(c, h.svc.ConfigMapDetail)
}

func (h *ConfigHandler) DeleteConfigMap(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteConfigMap)
}

// Secrets
func (h *ConfigHandler) ListSecrets(c *gin.Context) {
	handleQuery(c, h.svc.ListSecrets)
}

func (h *ConfigHandler) SecretDetail(c *gin.Context) {
	handleQuery(c, h.svc.SecretDetail)
}

func (h *ConfigHandler) DeleteSecret(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteSecret)
}

// shared apply
func (h *ConfigHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}
