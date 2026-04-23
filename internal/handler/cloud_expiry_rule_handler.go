package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type CloudExpiryRuleHandler struct {
	svc      *service.CloudExpiryRuleService
	alertSvc *service.AlertService
}

func NewCloudExpiryRuleHandler(svc *service.CloudExpiryRuleService, alertSvc *service.AlertService) *CloudExpiryRuleHandler {
	return &CloudExpiryRuleHandler{svc: svc, alertSvc: alertSvc}
}

func (h *CloudExpiryRuleHandler) List(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.CloudExpiryRuleListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		}, nil
	})
}

func (h *CloudExpiryRuleHandler) Create(c *gin.Context) {
	handleJSON(c, h.svc.Create)
}

func (h *CloudExpiryRuleHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.CloudExpiryRuleUpsertRequest) (*model.CloudExpiryRule, error) {
		return h.svc.Update(ctx, id, req)
	})
}

func (h *CloudExpiryRuleHandler) Delete(c *gin.Context) {
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

func (h *CloudExpiryRuleHandler) EvaluateNow(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "ok"}, func(ctx context.Context, _ map[string]any) error {
		return h.alertSvc.EvaluateCloudExpiryRulesNow(ctx)
	})
}

