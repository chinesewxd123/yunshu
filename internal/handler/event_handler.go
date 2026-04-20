package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	svc *service.K8sEventService
}

// NewEventHandler 创建相关逻辑。
func NewEventHandler(svc *service.K8sEventService) *EventHandler {
	return &EventHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *EventHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}
