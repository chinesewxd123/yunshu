package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type IngressHandler struct {
	svc *service.K8sIngressService
}

// NewIngressHandler 创建相关逻辑。
func NewIngressHandler(svc *service.K8sIngressService) *IngressHandler {
	return &IngressHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}

// RestartNginxPods 处理对应的 HTTP 请求并返回统一响应。
func (h *IngressHandler) RestartNginxPods(c *gin.Context) {
	handleJSON(c, h.svc.RestartIngressNginxPods)
}

// ListClasses 查询列表对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) ListClasses(c *gin.Context) {
	handleQuery(c, h.svc.ListClasses)
}

// DetailClass 查询详情对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) DetailClass(c *gin.Context) {
	handleQuery(c, h.svc.DetailClass)
}

// ApplyClass 提交申请对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) ApplyClass(c *gin.Context) {
	handleJSONOK(c, true, h.svc.ApplyClass)
}

// DeleteClass 删除对应的 HTTP 接口处理逻辑。
func (h *IngressHandler) DeleteClass(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteClass)
}
