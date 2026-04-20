package handler

import (
	"context"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginLogHandler struct {
	svc *service.LoginLogService
}

// NewLoginLogHandler 创建相关逻辑。
func NewLoginLogHandler(svc *service.LoginLogService) *LoginLogHandler {
	return &LoginLogHandler{svc: svc}
}

// List godoc
// @Summary List login logs
// @Tags LoginLog
// @Produce json
// @Security BearerAuth
// @Param username query string false "Username filter"
// @Param status query int false "1 success 0 fail"
// @Param source query string false "password or email"
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Body{data=pagination.Result[model.LoginLog]}
// @Router /api/v1/login-logs [get]
func (h *LoginLogHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

type idsRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchDelete godoc
// @Summary Batch delete login logs
// @Tags LoginLog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body idsRequest true "IDs"
// @Success 200 {object} response.Body{data=MessageData}
// @Router /api/v1/login-logs/delete [post]
func (h *LoginLogHandler) BatchDelete(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "deleted"}, func(ctx context.Context, req idsRequest) error {
		return h.svc.DeleteBatch(ctx, req.IDs)
	})
}

// Delete godoc
// @Summary Delete login log
// @Tags LoginLog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Log ID"
// @Success 200 {object} response.Body{data=MessageData}
// @Router /api/v1/login-logs/{id} [delete]
func (h *LoginLogHandler) Delete(c *gin.Context) {
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

// Export godoc
// @Summary Export login logs to Excel
// @Tags LoginLog
// @Produce application/octet-stream
// @Security BearerAuth
// @Param username query string false "Username filter"
// @Param status query int false "1 success 0 fail"
// @Param source query string false "password or email"
// @Router /api/v1/login-logs/export [get]
func (h *LoginLogHandler) Export(c *gin.Context) {
	var q service.LoginLogListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=login_logs.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Status(http.StatusOK)
	if err := h.svc.Export(c.Request.Context(), q, c.Writer); err != nil {
		response.Error(c, err)
		return
	}
}
