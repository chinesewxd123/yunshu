package handler

import (
	"strconv"
	"strings"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sScopedPolicyHandler struct {
	svc *service.K8sScopedPolicyService
}

// NewK8sScopedPolicyHandler 创建相关逻辑。
func NewK8sScopedPolicyHandler(svc *service.K8sScopedPolicyService) *K8sScopedPolicyHandler {
	return &K8sScopedPolicyHandler{svc: svc}
}

// Actions 处理对应的 HTTP 请求并返回统一响应。
func (h *K8sScopedPolicyHandler) Actions(c *gin.Context) {
	response.Success(c, gin.H{"list": h.svc.ActionCatalog()})
}

// Paths 处理对应的 HTTP 请求并返回统一响应。
func (h *K8sScopedPolicyHandler) Paths(c *gin.Context) {
	response.Success(c, gin.H{"list": h.svc.PathCatalog()})
}

// Grant 处理对应的 HTTP 请求并返回统一响应。
func (h *K8sScopedPolicyHandler) Grant(c *gin.Context) {
	handleJSON(c, h.svc.Grant)
}

// ListByRole 查询列表对应的 HTTP 接口处理逻辑。
func (h *K8sScopedPolicyHandler) ListByRole(c *gin.Context) {
	// 前端首屏初始化阶段可能先请求一次未带 role_id，
	// 这里兼容返回空列表，避免整页报 参数不合法。
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
