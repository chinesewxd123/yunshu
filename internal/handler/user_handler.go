package handler

import (
	"strconv"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create godoc
// @Summary Create user
// @Description Create a new user and optionally bind roles.
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.UserCreateRequest true "Create user request"
// @Success 201 {object} response.Body{data=service.UserDetailResponse} "created"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req service.UserCreateRequest
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
// @Summary Update user
// @Description Update user nickname, password or status.
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body service.UserUpdateRequest true "Update user request"
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req service.UserUpdateRequest
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
// @Summary Delete user
// @Description Delete a user by ID.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
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
// @Summary Get user detail
// @Description Get user detail by ID.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) Detail(c *gin.Context) {
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
// @Summary List users
// @Description List users with optional keyword and pagination filters.
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param keyword query string false "Search keyword"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.Body{data=UserPageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var query service.UserListQuery
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

// AssignRoles godoc
// @Summary Assign roles to user
// @Description Replace all user roles with the provided role ID list.
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body service.UserAssignRolesRequest true "Assign roles request"
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/users/{id}/roles [put]
func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req service.UserAssignRolesRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.service.AssignRoles(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func parseUintParam(c *gin.Context, key string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, apperror.BadRequest("invalid parameter")
	}
	return uint(id), nil
}
