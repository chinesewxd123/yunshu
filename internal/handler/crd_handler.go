package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type CRDHandler struct {
	svc *service.K8sCRDService
}

func NewCRDHandler(svc *service.K8sCRDService) *CRDHandler {
	return &CRDHandler{svc: svc}
}

func (h *CRDHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *CRDHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *CRDHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *CRDHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
