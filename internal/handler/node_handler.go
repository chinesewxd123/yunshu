package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NodeHandler struct {
	svc *service.K8sNodeService
}

// NewNodeHandler 创建相关逻辑。
func NewNodeHandler(svc *service.K8sNodeService) *NodeHandler {
	return &NodeHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *NodeHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *NodeHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// SetSchedulability 设置对应的 HTTP 接口处理逻辑。
func (h *NodeHandler) SetSchedulability(c *gin.Context) {
	handleJSONOK(c, gin.H{"ok": true}, h.svc.SetSchedulability)
}

// ReplaceTaints 处理对应的 HTTP 请求并返回统一响应。
func (h *NodeHandler) ReplaceTaints(c *gin.Context) {
	handleJSONOK(c, gin.H{"ok": true}, h.svc.ReplaceTaints)
}
