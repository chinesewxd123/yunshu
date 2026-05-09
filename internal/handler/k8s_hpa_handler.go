package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sHPAHandler struct {
	svc *service.K8sHPAService
}

func NewK8sHPAHandler(svc *service.K8sHPAService) *K8sHPAHandler {
	return &K8sHPAHandler{svc: svc}
}

func (h *K8sHPAHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

func (h *K8sHPAHandler) Detail(c *gin.Context) {
	ServeQuery(c, h.svc.Detail)
}

func (h *K8sHPAHandler) Apply(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.Apply)
}

func (h *K8sHPAHandler) Delete(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.Delete)
}
