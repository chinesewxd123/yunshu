package handler

import (
	"errors"
	"strconv"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
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
	// 如果不是 super-admin，需要从菜单树中移除仅管理员可见项
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Success(c, list)
		return
	}

	isSuper := false
	for _, rc := range user.RoleCodes {
		if strings.TrimSpace(rc) == "super-admin" {
			isSuper = true
			break
		}
	}

	if isSuper {
		response.Success(c, list)
		return
	}

	// 递归过滤 AdminOnly 项
	var filter func([]model.Menu) []model.Menu
	filter = func(items []model.Menu) []model.Menu {
		var out []model.Menu
		for _, it := range items {
			if it.AdminOnly {
				continue
			}
			if len(it.Children) > 0 {
				it.Children = filter(it.Children)
			}
			out = append(out, it)
		}
		return out
	}

	filtered := filter(list)
	response.Success(c, filtered)
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
