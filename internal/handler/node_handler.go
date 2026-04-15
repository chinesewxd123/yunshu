package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NodeHandler struct {
	svc *service.K8sNodeService
}

func NewNodeHandler(svc *service.K8sNodeService) *NodeHandler {
	return &NodeHandler{svc: svc}
}

func (h *NodeHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *NodeHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *NodeHandler) SetSchedulability(c *gin.Context) {
	handleJSONOK(c, gin.H{"ok": true}, h.svc.SetSchedulability)
}

func (h *NodeHandler) ReplaceTaints(c *gin.Context) {
	handleJSONOK(c, gin.H{"ok": true}, h.svc.ReplaceTaints)
}
