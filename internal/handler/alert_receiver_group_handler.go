package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/response"
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
	handleQuery(c, func(ctx context.Context, q service.AlertReceiverGroupListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertReceiverGroupHandler) Create(c *gin.Context) {
	handleJSON(c, h.svc.Create)
}

func (h *AlertReceiverGroupHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertReceiverGroupUpsertRequest) (*model.AlertReceiverGroup, error) {
		return h.svc.Update(ctx, id, req)
	})
}

func (h *AlertReceiverGroupHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

