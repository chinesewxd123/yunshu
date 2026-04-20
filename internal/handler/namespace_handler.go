package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NamespaceHandler struct {
	svc *service.K8sNamespaceService
}

// NewNamespaceHandler 创建相关逻辑。
func NewNamespaceHandler(svc *service.K8sNamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *NamespaceHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
