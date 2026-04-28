package handler

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// AlertPlatformHandler 告警平台：数据源、静默、监控规则、处理人、值班、PromQL。
type AlertPlatformHandler struct {
	ds      *service.AlertDatasourceService
	silence *service.AlertSilenceService
	rules   *service.AlertMonitorRuleService
	assign  *service.AlertRuleAssigneeService
	duty    *service.AlertDutyService
}

func NewAlertPlatformHandler(
	ds *service.AlertDatasourceService,
	silence *service.AlertSilenceService,
	rules *service.AlertMonitorRuleService,
	assign *service.AlertRuleAssigneeService,
	duty *service.AlertDutyService,
) *AlertPlatformHandler {
	return &AlertPlatformHandler{ds: ds, silence: silence, rules: rules, assign: assign, duty: duty}
}

func alertPlatformUserID(c *gin.Context) uint {
	if u, ok := auth.CurrentUserFromContext(c); ok {
		return u.ID
	}
	return 0
}

func (h *AlertPlatformHandler) ListDatasources(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertDatasourceListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.ds.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateDatasource(c *gin.Context) {
	handleJSON(c, h.ds.Create)
}

func (h *AlertPlatformHandler) UpdateDatasource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertDatasourceUpsertRequest) (any, error) {
		return h.ds.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteDatasource(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.ds.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) PromQuery(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.PromQueryRequest) (any, error) {
		raw, err := h.ds.PromQuery(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return gin.H{"data": raw}, nil
	})
}

func (h *AlertPlatformHandler) PromQueryRange(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.PromQueryRangeRequest) (any, error) {
		raw, err := h.ds.PromQueryRange(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return gin.H{"data": raw}, nil
	})
}

func (h *AlertPlatformHandler) PromActiveAlerts(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	raw, err := h.ds.PrometheusActiveAlerts(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"data": raw})
}

func (h *AlertPlatformHandler) ListSilences(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertSilenceListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.silence.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateSilence(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.AlertSilenceUpsertRequest) (any, error) {
		return h.silence.Create(ctx, alertPlatformUserID(c), req)
	})
}

func (h *AlertPlatformHandler) CreateSilenceBatch(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.AlertSilenceBatchRequest) (gin.H, error) {
		n, err := h.silence.CreateBatch(ctx, alertPlatformUserID(c), req)
		if err != nil {
			return nil, err
		}
		return gin.H{"created": n}, nil
	})
}

func (h *AlertPlatformHandler) UpdateSilence(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertSilenceUpsertRequest) (any, error) {
		return h.silence.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteSilence(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.silence.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) ListMonitorRules(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertMonitorRuleListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.rules.List(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateMonitorRule(c *gin.Context) {
	handleJSON(c, h.rules.Create)
}

func (h *AlertPlatformHandler) UpdateMonitorRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertMonitorRuleUpsertRequest) (any, error) {
		return h.rules.Update(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteMonitorRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.rules.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *AlertPlatformHandler) GetMonitorRuleAssignees(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.assign.ListByRule(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *AlertPlatformHandler) UpsertMonitorRuleAssignees(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertRuleAssigneeUpsertRequest) (any, error) {
		return h.assign.UpsertPrimary(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) ListDutyBlocks(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertDutyBlockListQuery) (gin.H, error) {
		list, total, page, pageSize, err := h.duty.ListBlocks(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"items": list, "list": list, "total": total, "page": page, "page_size": pageSize}, nil
	})
}

func (h *AlertPlatformHandler) CreateDutyBlock(c *gin.Context) {
	handleJSON(c, h.duty.CreateBlock)
}

func (h *AlertPlatformHandler) UpdateDutyBlock(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertDutyBlockUpsertRequest) (any, error) {
		return h.duty.UpdateBlock(ctx, id, req)
	})
}

func (h *AlertPlatformHandler) DeleteDutyBlock(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.duty.DeleteBlock(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
