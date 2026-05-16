package handler

import (
	"context"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// AlertInhibitionHandler 告警抑制规则 HTTP API（对齐 Alertmanager inhibit_rules 语义）。
type AlertInhibitionHandler struct {
	svc *service.AlertInhibitionService
}

func NewAlertInhibitionHandler(svc *service.AlertInhibitionService) *AlertInhibitionHandler {
	return &AlertInhibitionHandler{svc: svc}
}

// List godoc
// @Summary 查询告警抑制规则列表
// @Description 分页查询抑制规则。规则在入站 firing 时生效：源告警匹配后写入 Redis，目标告警匹配且 equal 标签一致则被抑制。
// @Tags alert-inhibition
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "名称/描述关键词"
// @Param enabled query bool false "是否启用"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/alerts/inhibition-rules [get]
func (h *AlertInhibitionHandler) List(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.AlertInhibitionRuleListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.svc.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

// Create godoc
// @Summary 创建告警抑制规则
// @Tags alert-inhibition
// @Accept json
// @Produce json
// @Param body body service.AlertInhibitionRuleUpsertRequest true "规则"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/alerts/inhibition-rules [post]
func (h *AlertInhibitionHandler) Create(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.AlertInhibitionRuleUpsertRequest) (any, error) {
		return h.svc.Create(ctx, req)
	})
}

// Update godoc
// @Summary 更新告警抑制规则
// @Tags alert-inhibition
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param body body service.AlertInhibitionRuleUpsertRequest true "规则"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/alerts/inhibition-rules/{id} [put]
func (h *AlertInhibitionHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertInhibitionRuleUpsertRequest) (any, error) {
		return h.svc.Update(ctx, id, req)
	})
}

// Delete godoc
// @Summary 删除告警抑制规则
// @Tags alert-inhibition
// @Param id path int true "规则 ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/alerts/inhibition-rules/{id} [delete]
func (h *AlertInhibitionHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// RefreshCache godoc
// @Summary 刷新抑制规则内存缓存
// @Tags alert-inhibition
// @Router /api/v1/alerts/inhibition-rules/refresh-cache [post]
func (h *AlertInhibitionHandler) RefreshCache(c *gin.Context) {
	if err := h.svc.RefreshCache(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"refreshed": true})
}
