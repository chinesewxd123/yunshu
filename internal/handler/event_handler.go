package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	svc *service.K8sEventService
}

func NewEventHandler(svc *service.K8sEventService) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}
