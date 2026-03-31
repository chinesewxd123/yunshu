package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	service *service.PermissionService
}

func NewPermissionHandler(service *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

// Create godoc
// @Summary Create permission
// @Description Create a new permission resource and action pair.
// @Tags Permission
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PermissionCreateRequest true "Create permission request"
// @Success 201 {object} response.Body{data=service.PermissionItem} "created"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var req service.PermissionCreateRequest
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
// @Summary Update permission
// @Description Update permission name, resource, action or description.
// @Tags Permission
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Permission ID"
// @Param request body service.PermissionUpdateRequest true "Update permission request"
// @Success 200 {object} response.Body{data=service.PermissionItem} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "permission not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.PermissionUpdateRequest
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
// @Summary Delete permission
// @Description Delete a permission by ID.
// @Tags Permission
// @Produce json
// @Security BearerAuth
// @Param id path int true "Permission ID"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "permission not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
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
// @Summary Get permission detail
// @Description Get permission detail by ID.
// @Tags Permission
// @Produce json
// @Security BearerAuth
// @Param id path int true "Permission ID"
// @Success 200 {object} response.Body{data=service.PermissionItem} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "permission not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/permissions/{id} [get]
func (h *PermissionHandler) Detail(c *gin.Context) {
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
// @Summary List permissions
// @Description List permissions with optional keyword and pagination filters.
// @Tags Permission
// @Produce json
// @Security BearerAuth
// @Param keyword query string false "Search keyword"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Body{data=PermissionPageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	var query service.PermissionListQuery
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
