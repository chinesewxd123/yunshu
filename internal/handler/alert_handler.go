package handler

import (
	"context"
	"log/slog"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
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
	ServeQuery(c, func(ctx context.Context, q service.AlertChannelListQuery) (gin.H, error) {
		list, err := h.svc.ListChannels(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// CreateChannel 创建对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	ServeJSON(c, h.svc.CreateChannel)
}

// UpdateChannel 更新对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.AlertChannelUpsertRequest) (*model.AlertChannel, error) {
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
	ServeJSONOK(c, gin.H{"message": "test sent"}, func(ctx context.Context, req service.AlertTestRequest) error {
		return h.svc.TestChannel(ctx, id, req)
	})
}

// PreviewChannelTemplate 预览通道模板渲染结果。
func (h *AlertHandler) PreviewChannelTemplate(c *gin.Context) {
	ServeJSON(c, h.svc.PreviewChannelTemplate)
}

// ListEvents 查询列表对应的 HTTP 接口处理逻辑。
func (h *AlertHandler) ListEvents(c *gin.Context) {
	var q service.AlertEventListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(err.Error()))
		return
	}
	list, total, page, pageSize, err := h.svc.ListEvents(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":     list,
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
		response.Error(c, constants.ErrAlertWebhookTokenInvalid)
		return
	}

	// 先快速返回 202 Accepted，后台异步处理入队/处理逻辑，避免阻塞上游 Alertmanager
	var payload service.AlertManagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(bindErrorMessage(err)))
		return
	}

	// 异步处理：调用 Service 层统一入口（内部会根据配置选择入队或同步处理）
	go func(p service.AlertManagerPayload) {
		if err := h.svc.ReceiveAlertmanager(context.Background(), p); err != nil {
			slog.Error("alertmanager webhook processing failed",
				slog.String("status", p.Status),
				slog.String("receiver", p.Receiver),
				slog.Int("alerts", len(p.Alerts)),
				slog.Any("error", err))
		}
	}(payload)

	c.JSON(202, gin.H{"message": "accepted"})
}
