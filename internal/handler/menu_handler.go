package handler

import (
	"errors"
	"strconv"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MenuHandler struct {
	service *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{service: svc}
}

func (h *MenuHandler) Tree(c *gin.Context) {
	list, err := h.service.Tree(c.Request.Context())
	if err != nil {
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	response.Success(c, list)
}

func (h *MenuHandler) Create(c *gin.Context) {
	var req service.MenuCreatePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	response.Success(c, data)
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, apperror.BadRequest("invalid id"))
		return
	}
	var req service.MenuUpdatePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, apperror.NotFound("menu not found"))
			return
		}
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	response.Success(c, data)
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, apperror.BadRequest("invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, apperror.NotFound("menu not found"))
			return
		}
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
