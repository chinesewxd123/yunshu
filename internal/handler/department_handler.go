package handler

import (
	"context"
	"errors"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DepartmentHandler struct {
	service *service.DepartmentService
}

func NewDepartmentHandler(service *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

func (h *DepartmentHandler) Tree(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	data, err := h.service.TreeByActor(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *DepartmentHandler) Detail(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.service.DetailByActor(c.Request.Context(), user, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleJSONCreated(c, func(ctx context.Context, req service.DepartmentCreateRequest) (*service.DepartmentDetailResponse, error) {
		return h.service.CreateByActor(ctx, user, req)
	})
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.DepartmentUpdateRequest) (*service.DepartmentDetailResponse, error) {
		return h.service.UpdateByActor(ctx, user, id, req)
	})
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = h.service.DeleteByActor(c.Request.Context(), user, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, apperror.NotFound("部门不存在"))
			return
		}
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
