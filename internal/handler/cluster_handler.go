package handler

import (
	"context"
	"net/http"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ClusterHandler struct {
	svc *service.K8sClusterService
}

// NewClusterHandler 创建相关逻辑。
func NewClusterHandler(svc *service.K8sClusterService) *ClusterHandler {
	return &ClusterHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

// Create 创建对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) Create(c *gin.Context) {
	ServeJSON201(c, h.svc.Create)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) Detail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Update 更新对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) Update(c *gin.Context) {
	ServePatch(c, h.svc.Update, "")
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) Delete(c *gin.Context) {
	ServeDelete(c, h.svc.Delete, "")
}

// Status 处理对应的 HTTP 请求并返回统一响应。
func (h *ClusterHandler) Status(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	data, err := h.svc.Status(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Namespaces 处理对应的 HTTP 请求并返回统一响应。
func (h *ClusterHandler) Namespaces(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	user, ok := auth.CurrentUserFromContext(c)
	var pack *k8sauth.PrincipalPack
	if ok && user != nil {
		p := k8sauth.PackFromCurrentUser(user)
		pack = &p
	}
	list, err := h.svc.ListNamespaces(c.Request.Context(), id, pack)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// ComponentStatuses 处理对应的 HTTP 请求并返回统一响应。
func (h *ClusterHandler) ComponentStatuses(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListComponentStatuses(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// SetStatus 设置对应的 HTTP 接口处理逻辑。
func (h *ClusterHandler) SetStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	ServeJSON(c, func(ctx context.Context, req service.K8sClusterSetStatusRequest) (*service.K8sClusterItem, error) {
		return h.svc.SetStatus(ctx, id, req.Status)
	})
}

// Not used now, left for swagger generation.
var _ = http.MethodGet
