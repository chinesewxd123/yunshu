package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	svc *service.AlertService
}

// NewAlertHandler 创建相关逻辑。
func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// ListChannels 查询列表对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) ListChannels(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.AlertChannelListQuery) (gin.H, error) {
		list, err := h.svc.ListChannels(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// CreateChannel 创建对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	handleJSON(c, h.svc.CreateChannel)
}

// UpdateChannel 更新对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AlertChannelUpsertRequest) (*model.AlertChannel, error) {
		return h.svc.UpdateChannel(ctx, id, req)
	})
}

// DeleteChannel 删除对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteChannel(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// TestChannel 测试对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) TestChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSONOK(c, gin.H{"message": "test sent"}, func(ctx context.Context, req service.AlertTestRequest) error {
		return h.svc.TestChannel(ctx, id, req)
	})
}

// ListEvents 查询列表对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) ListEvents(c *gin.Context) {
	var q service.AlertEventListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	list, total, page, pageSize, err := h.svc.ListEvents(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// HistoryStats 处理对应的 HTTP 请求并返回统一响应。
func (h *AlertHandler) HistoryStats(c *gin.Context) {
	stats, err := h.svc.HistoryStats(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

// ReceiveAlertmanager 处理对应的 HTTP 请求并返回统一响应。
func (h *AlertHandler) ReceiveAlertmanager(c *gin.Context) {
	token := c.GetHeader("X-Alert-Token")
	if token == "" {
		token = c.GetHeader("X-Webhook-Token")
	}
	if token == "" {
		token = c.GetHeader("Authorization")
	}
	if token == "" {
		token = c.Query("token")
	}
	if !h.svc.ValidateWebhookToken(token) {
		response.Error(c, apperror.Forbidden("告警回调令牌无效"))
		return
	}
	handleJSONOK(c, gin.H{"message": "accepted"}, h.svc.ReceiveAlertmanager)
}
