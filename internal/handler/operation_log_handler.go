package handler

import (
	"context"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"net/http"

	"github.com/gin-gonic/gin"
)

type OperationLogHandler struct {
	svc *service.OperationLogService
}

// NewOperationLogHandler 创建相关逻辑。
func NewOperationLogHandler(svc *service.OperationLogService) *OperationLogHandler {
	return &OperationLogHandler{svc: svc}
}

// List godoc
// @Summary List operation logs
// @Tags OperationLog
// @Produce json
// @Security BearerAuth
// @Param method query string false "HTTP method"
// @Param path query string false "Path contains"
// @Param status_code query int false "HTTP status"
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Body
// @Router /api/v1/operation-logs [get]
func (h *OperationLogHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

// BatchDelete godoc
// @Summary Batch delete operation logs
// @Tags OperationLog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body idsRequest true "IDs"
// @Success 200 {object} response.Body{data=MessageData}
// @Router /api/v1/operation-logs/delete [post]
func (h *OperationLogHandler) BatchDelete(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "deleted"}, func(ctx context.Context, req idsRequest) error {
		return h.svc.DeleteBatch(ctx, req.IDs)
	})
}

// Delete godoc
// @Summary Delete operation log
// @Tags OperationLog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Log ID"
// @Success 200 {object} response.Body{data=MessageData}
// @Router /api/v1/operation-logs/{id} [delete]
func (h *OperationLogHandler) Delete(c *gin.Context) {
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
// @Summary Export operation logs to Excel
// @Tags OperationLog
// @Produce application/octet-stream
// @Security BearerAuth
// @Param method query string false "HTTP method"
// @Param path query string false "Path contains"
// @Param status_code query int false "HTTP status"
// @Router /api/v1/operation-logs/export [get]
func (h *OperationLogHandler) Export(c *gin.Context) {
	var q service.OperationLogListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=operation_logs.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Status(http.StatusOK)
	if err := h.svc.Export(c.Request.Context(), q, c.Writer); err != nil {
		response.Error(c, err)
		return
	}
}
