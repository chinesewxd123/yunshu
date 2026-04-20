package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type OverviewHandler struct {
	svc *service.OverviewService
}

// NewOverviewHandler 创建相关逻辑。
func NewOverviewHandler(svc *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// Get 获取对应的 HTTP 接口处理逻辑。
func (h *OverviewHandler) Get(c *gin.Context) {
	data, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Trends 处理对应的 HTTP 请求并返回统一响应。
func (h *OverviewHandler) Trends(c *gin.Context) {
	data, err := h.svc.Trends(c.Request.Context(), 7)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}
