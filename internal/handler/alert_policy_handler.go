package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertPolicyHandler struct {
	svc *service.AlertPolicyService
}

func NewAlertPolicyHandler(svc *service.AlertPolicyService) *AlertPolicyHandler {
	return &AlertPolicyHandler{svc: svc}
}

func (h *AlertPolicyHandler) List(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertPolicyListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPolicyHandler) Create(c *gin.Context) {
	handleJSON(c, h.svc.Create)
}

func (h *AlertPolicyHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertPolicyUpsertRequest) (*model.AlertPolicy, error) {
		return h.svc.Update(ctx, id, req)
	})
}

func (h *AlertPolicyHandler) Delete(c *gin.Context) {
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
