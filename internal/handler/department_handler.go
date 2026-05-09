package handler

import (
	"context"
	"errors"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

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
		response.Error(c, constants.ErrUnauthorized)
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
		response.Error(c, constants.ErrUnauthorized)
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
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req service.DepartmentCreateRequest) (*service.DepartmentDetailResponse, error) {
		return h.service.CreateByActor(ctx, user, req)
	})
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.DepartmentUpdateRequest) (*service.DepartmentDetailResponse, error) {
		return h.service.UpdateByActor(ctx, user, id, req)
	})
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = h.service.DeleteByActor(c.Request.Context(), user, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, constants.ErrDepartmentNotFound)
			return
		}
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
