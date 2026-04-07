package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type OverviewHandler struct {
	svc *service.OverviewService
}

func NewOverviewHandler(svc *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

func (h *OverviewHandler) Get(c *gin.Context) {
	data, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}
