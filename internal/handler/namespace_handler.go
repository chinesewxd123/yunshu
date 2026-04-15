package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NamespaceHandler struct {
	svc *service.K8sNamespaceService
}

func NewNamespaceHandler(svc *service.K8sNamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

func (h *NamespaceHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *NamespaceHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *NamespaceHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *NamespaceHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
