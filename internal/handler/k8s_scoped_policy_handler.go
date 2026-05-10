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

// GrantPreset 按 readonly / readonly_exec / admin 档位写入集群授权（DB）。
func (h *K8sScopedPolicyHandler) GrantPreset(c *gin.Context) {
	ServeJSON(c, h.svc.GrantPreset)
}

// DeleteClusterGrant 删除一条集群档位记录。
func (h *K8sScopedPolicyHandler) DeleteClusterGrant(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteClusterGrant(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListByRole 查询集群档位列表；支持 role_id、user_id、group_id（仅第一个非零参数生效，优先级 role > user > group）。
func (h *K8sScopedPolicyHandler) ListByRole(c *gin.Context) {
	parseID := func(key string) uint {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" || strings.EqualFold(raw, "undefined") || strings.EqualFold(raw, "null") || raw == "0" {
			return 0
		}
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return 0
		}
		return uint(parsed)
	}
	roleID := parseID("role_id")
	userID := parseID("user_id")
	groupID := parseID("group_id")
	if roleID == 0 && userID == 0 && groupID == 0 {
		response.Success(c, gin.H{"list": []service.K8sClusterAccessItem{}})
		return
	}
	list, err := h.svc.ListClusterGrants(c.Request.Context(), roleID, userID, groupID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}
