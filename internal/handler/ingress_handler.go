package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type IngressHandler struct {
	svc *service.K8sIngressService
}

func NewIngressHandler(svc *service.K8sIngressService) *IngressHandler {
	return &IngressHandler{svc: svc}
}

func (h *IngressHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *IngressHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *IngressHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *IngressHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}

func (h *IngressHandler) RestartNginxPods(c *gin.Context) {
	handleJSON(c, h.svc.RestartIngressNginxPods)
}

func (h *IngressHandler) ListClasses(c *gin.Context) {
	handleQuery(c, h.svc.ListClasses)
}

func (h *IngressHandler) DetailClass(c *gin.Context) {
	handleQuery(c, h.svc.DetailClass)
}

func (h *IngressHandler) ApplyClass(c *gin.Context) {
	handleJSONOK(c, true, h.svc.ApplyClass)
}

func (h *IngressHandler) DeleteClass(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteClass)
}
