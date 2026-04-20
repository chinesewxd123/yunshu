package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type PolicyHandler struct {
	service *service.PolicyService
}

// NewPolicyHandler 创建相关逻辑。
func NewPolicyHandler(service *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{service: service}
}

// List godoc
// @Summary List policies
// @Description List current role-permission policy bindings.
// @Tags Policy
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]service.PolicyItemResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/policies [get]
func (h *PolicyHandler) List(c *gin.Context) {
	data, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Grant godoc
// @Summary Grant policy
// @Description Bind one permission to one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Grant policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/policies [post]
func (h *PolicyHandler) Grant(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "granted"}, h.service.Grant)
}

// Revoke godoc
// @Summary Revoke policy
// @Description Remove one permission from one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Revoke policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/policies [delete]
func (h *PolicyHandler) Revoke(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "revoked"}, h.service.Revoke)
}
