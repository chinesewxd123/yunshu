package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type CRHandler struct {
	svc *service.K8sCRService
}

func NewCRHandler(svc *service.K8sCRService) *CRHandler {
	return &CRHandler{svc: svc}
}

func (h *CRHandler) ListResources(c *gin.Context) {
	handleQuery(c, h.svc.ListResources)
}

func (h *CRHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *CRHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *CRHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *CRHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
