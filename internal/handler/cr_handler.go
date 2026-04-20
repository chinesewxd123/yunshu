package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type CRHandler struct {
	svc *service.K8sCRService
}

// NewCRHandler 创建相关逻辑。
func NewCRHandler(svc *service.K8sCRService) *CRHandler {
	return &CRHandler{svc: svc}
}

// ListResources 查询列表对应的 HTTP 接口处理逻辑。
func (h *CRHandler) ListResources(c *gin.Context) {
	handleQuery(c, h.svc.ListResources)
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *CRHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *CRHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *CRHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *CRHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
