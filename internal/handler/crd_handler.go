package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type CRDHandler struct {
	svc *service.K8sCRDService
}

// NewCRDHandler 创建相关逻辑。
func NewCRDHandler(svc *service.K8sCRDService) *CRDHandler {
	return &CRDHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *CRDHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *CRDHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *CRDHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *CRDHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
