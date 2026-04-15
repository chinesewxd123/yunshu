package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ServiceResourceHandler struct {
	svc *service.K8sServiceResourceService
}

func NewServiceResourceHandler(svc *service.K8sServiceResourceService) *ServiceResourceHandler {
	return &ServiceResourceHandler{svc: svc}
}

func (h *ServiceResourceHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *ServiceResourceHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *ServiceResourceHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *ServiceResourceHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
