package handler

import (
	"context"

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
	ServeQuery(c, func(ctx context.Context, q service.CloudExpiryRuleListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"items":     list,
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		}, nil
	})
}

func (h *CloudExpiryRuleHandler) Create(c *gin.Context) {
	ServeJSON(c, h.svc.Create)
}

func (h *CloudExpiryRuleHandler) Update(c *gin.Context) {
	ServePatch(c, h.svc.Update, "")
}

func (h *CloudExpiryRuleHandler) Delete(c *gin.Context) {
	ServeDelete(c, h.svc.Delete, "")
}

func (h *CloudExpiryRuleHandler) EvaluateNow(c *gin.Context) {
	ServeJSONOK(c, gin.H{"message": "ok"}, func(ctx context.Context, _ map[string]any) error {
		return h.alertSvc.EvaluateCloudExpiryRulesNow(ctx)
	})
}
