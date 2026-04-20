package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ServiceResourceHandler struct {
	svc *service.K8sServiceResourceService
}

// NewServiceResourceHandler 创建相关逻辑。
func NewServiceResourceHandler(svc *service.K8sServiceResourceService) *ServiceResourceHandler {
	return &ServiceResourceHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *ServiceResourceHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *ServiceResourceHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *ServiceResourceHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *ServiceResourceHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
