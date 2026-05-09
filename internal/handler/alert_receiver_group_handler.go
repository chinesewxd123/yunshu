package handler

import (
	"context"

	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertReceiverGroupHandler struct {
	svc *service.AlertReceiverGroupService
}

func NewAlertReceiverGroupHandler(svc *service.AlertReceiverGroupService) *AlertReceiverGroupHandler {
	return &AlertReceiverGroupHandler{svc: svc}
}

func (h *AlertReceiverGroupHandler) List(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertReceiverGroupListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertReceiverGroupHandler) Create(c *gin.Context) {
	ServeJSON(c, h.svc.Create)
}

func (h *AlertReceiverGroupHandler) Update(c *gin.Context) {
	ServePatch(c, h.svc.Update, "")
}

func (h *AlertReceiverGroupHandler) Delete(c *gin.Context) {
	ServeDelete(c, h.svc.Delete, "")
}
