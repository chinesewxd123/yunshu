package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *service.RoleService
}

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
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req service.RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, data)
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
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "role not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.RoleUpdateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
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
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "role not found"
// @Failure 500 {object} response.Body "internal server error"
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
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "role not found"
// @Failure 500 {object} response.Body "internal server error"
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
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	var query service.RoleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}
