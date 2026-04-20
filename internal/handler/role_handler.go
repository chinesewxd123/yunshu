package handler

import (
	"context"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *service.RoleService
}

// NewRoleHandler 创建相关逻辑。
func NewRoleHandler(service *service.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

// Create godoc
// @Summary Create role
// @Description Create a new role.
// @Tags Role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.RoleCreateRequest true "Create role request"
// @Success 201 {object} response.Body{data=service.RoleItem} "created"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	handleJSONCreated(c, h.service.Create)
}

// Update godoc
// @Summary Update role
// @Description Update role name, code, description or status.
// @Tags Role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param request body service.RoleUpdateRequest true "Update role request"
// @Success 200 {object} response.Body{data=service.RoleItem} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "角色不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.RoleUpdateRequest) (*service.RoleItem, error) {
		return h.service.Update(ctx, id, req)
	})
}

// Delete godoc
// @Summary Delete role
// @Description Delete a role by ID.
// @Tags Role
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "角色不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// Detail godoc
// @Summary Get role detail
// @Description Get role detail by ID.
// @Tags Role
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} response.Body{data=service.RoleItem} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "角色不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) Detail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.service.Detail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// List godoc
// @Summary List roles
// @Description List roles with optional keyword and pagination filters.
// @Tags Role
// @Produce json
// @Security BearerAuth
// @Param keyword query string false "Search keyword"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Body{data=RolePageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	handleQuery(c, h.service.List)
}
