package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	svc *service.K8sConfigService
}

// NewConfigHandler 创建相关逻辑。
func NewConfigHandler(svc *service.K8sConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// ConfigMaps
func (h *ConfigHandler) ListConfigMaps(c *gin.Context) {
	handleQuery(c, h.svc.ListConfigMaps)
}

// ConfigMapDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *ConfigHandler) ConfigMapDetail(c *gin.Context) {
	handleQuery(c, h.svc.ConfigMapDetail)
}

// DeleteConfigMap 删除对应的 HTTP 接口处理逻辑。
func (h *ConfigHandler) DeleteConfigMap(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteConfigMap)
}

// Secrets
func (h *ConfigHandler) ListSecrets(c *gin.Context) {
	handleQuery(c, h.svc.ListSecrets)
}

// SecretDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *ConfigHandler) SecretDetail(c *gin.Context) {
	handleQuery(c, h.svc.SecretDetail)
}

// DeleteSecret 删除对应的 HTTP 接口处理逻辑。
func (h *ConfigHandler) DeleteSecret(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteSecret)
}

// shared apply
func (h *ConfigHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}
