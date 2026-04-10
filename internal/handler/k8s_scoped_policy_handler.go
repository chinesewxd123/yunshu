package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type K8sScopedPolicyHandler struct {
	svc *service.K8sScopedPolicyService
}

func NewK8sScopedPolicyHandler(svc *service.K8sScopedPolicyService) *K8sScopedPolicyHandler {
	return &K8sScopedPolicyHandler{svc: svc}
}

func (h *K8sScopedPolicyHandler) Actions(c *gin.Context) {
	response.Success(c, gin.H{"list": h.svc.ActionCatalog()})
}

func (h *K8sScopedPolicyHandler) Paths(c *gin.Context) {
	response.Success(c, gin.H{"list": h.svc.PathCatalog()})
}

func (h *K8sScopedPolicyHandler) Grant(c *gin.Context) {
	var req service.K8sScopedPolicyGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.svc.Grant(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *K8sScopedPolicyHandler) ListByRole(c *gin.Context) {
	// 前端首屏初始化阶段可能先请求一次未带 role_id，
	// 这里兼容返回空列表，避免整页报 invalid parameter。
	raw := strings.TrimSpace(c.Query("role_id"))
	if raw == "" || strings.EqualFold(raw, "undefined") || strings.EqualFold(raw, "null") || raw == "0" {
		response.Success(c, gin.H{"list": []service.K8sScopedPolicyItem{}})
		return
	}
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		// 兼容前端传入异常值（例如 role_id=NaN），避免阻断页面渲染
		response.Success(c, gin.H{"list": []service.K8sScopedPolicyItem{}})
		return
	}
	roleID := uint(parsed)
	list, err := h.svc.ListByRole(c.Request.Context(), roleID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}
