package handler

import (
	"context"
	"strconv"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type RegistrationHandler struct {
	service *service.RegistrationService
}

// NewRegistrationHandler 创建相关逻辑。
func NewRegistrationHandler(svc *service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{service: svc}
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *RegistrationHandler) Apply(c *gin.Context) {
	handleJSONOK(c, nil, func(ctx context.Context, req service.ApplyRegisterRequest) error {
		if err := h.service.Apply(ctx, req); err != nil {
			if _, ok := apperror.IsAppError(err); ok {
				return err
			}
			return apperror.Internal(err.Error())
		}
		return nil
	})
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *RegistrationHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	var status *int
	if s := c.Query("status"); s != "" {
		v, err := strconv.Atoi(s)
		if err == nil {
			status = &v
		}
	}

	list, total, err := h.service.List(c.Request.Context(), keyword, status, page, pageSize)
	if err != nil {
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Review 处理对应的 HTTP 请求并返回统一响应。
func (h *RegistrationHandler) Review(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ReviewRequest) (gin.H, error) {
		if err := h.service.Review(ctx, id, user.ID, req); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		statusText := "approved"
		if req.Status == 2 {
			statusText = "rejected"
		}
		return gin.H{"message": "registration request has been " + statusText}, nil
	})
}
