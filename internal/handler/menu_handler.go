package handler

import (
	"context"
	"errors"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MenuHandler 菜单管理：树查询、增删改；非 super-admin 会过滤 AdminOnly 菜单。
type MenuHandler struct {
	service *service.MenuService
}

// NewMenuHandler 构造菜单处理器。
func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{service: svc}
}

// Tree 返回菜单树；非 super-admin 时移除仅管理员可见项。
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

// Create 创建菜单项。
func (h *MenuHandler) Create(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.MenuCreatePayload) (*model.Menu, error) {
		data, err := h.service.Create(ctx, req)
		if err != nil {
			return nil, apperror.Internal(err.Error())
		}
		return data, nil
	})
}

// Update 按 ID 更新菜单。
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.MenuUpdatePayload) (*model.Menu, error) {
		data, err := h.service.Update(ctx, id, req)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("菜单不存在")
			}
			return nil, apperror.Internal(err.Error())
		}
		return data, nil
	})
}

// Delete 按 ID 删除菜单。
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, apperror.NotFound("菜单不存在"))
			return
		}
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// BatchStatus 批量设置菜单状态（启用/停用）。
func (h *MenuHandler) BatchStatus(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "updated"}, func(ctx context.Context, req service.MenuBatchStatusPayload) error {
		if err := h.service.BatchSetStatus(ctx, req); err != nil {
			if _, ok := apperror.IsAppError(err); ok {
				return err
			}
			return apperror.Internal(err.Error())
		}
		return nil
	})
}
