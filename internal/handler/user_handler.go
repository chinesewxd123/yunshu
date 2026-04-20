package handler

import (
	"context"
	"net/http"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type UserHandler struct {
	service *service.UserService
}

// NewUserHandler 创建相关逻辑。
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleJSONCreated(c, func(ctx context.Context, req service.UserCreateRequest) (*service.UserDetailResponse, error) {
		return h.service.CreateByActor(ctx, user, req)
	})
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
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

	handleJSON(c, func(ctx context.Context, req service.UserUpdateRequest) (*service.UserDetailResponse, error) {
		return h.service.UpdateByActor(ctx, user, id, req)
	})
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) Detail(c *gin.Context) {
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleQuery(c, func(ctx context.Context, req service.UserListQuery) (*pagination.Result[service.UserDetailResponse], error) {
		return h.service.ListByActor(ctx, user, req)
	})
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/users/{id}/roles [put]
func (h *UserHandler) AssignRoles(c *gin.Context) {
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

	handleJSON(c, func(ctx context.Context, req service.UserAssignRolesRequest) (*service.UserDetailResponse, error) {
		return h.service.AssignRolesByActor(ctx, user, id, req)
	})
}

// Export godoc
// @Summary Export users to Excel
// @Tags User
// @Produce application/octet-stream
// @Security BearerAuth
// @Router /api/v1/users/export [get]
func (h *UserHandler) Export(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	users, err := h.service.ListAllByActor(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err)
		return
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	// header
	_ = f.SetSheetRow(sheet, "A1", &[]interface{}{"ID", "Username", "Nickname", "Email", "Status", "Department"})
	for i, u := range users {
		email := ""
		departmentName := ""
		if u.Email != nil {
			email = *u.Email
		}
		if u.Department != nil {
			departmentName = u.Department.Name
		}
		row := []interface{}{u.ID, u.Username, u.Nickname, email, int(u.Status), departmentName}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=users.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Status(http.StatusOK)
	_ = f.Write(c.Writer)
}

// ImportTemplate godoc
// @Summary Download user import template
// @Tags User
// @Produce application/octet-stream
// @Security BearerAuth
// @Router /api/v1/users/import-template [get]
func (h *UserHandler) ImportTemplate(c *gin.Context) {
	f, err := h.service.UsersImportTemplateExcel()
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=users-import-template.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Status(http.StatusOK)
	_ = f.Write(c.Writer)
}

// Import godoc
// @Summary Import users from Excel
// @Tags User
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file"
// @Security BearerAuth
// @Router /api/v1/users/import [post]
func (h *UserHandler) Import(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("文件上传失败"))
		return
	}
	defer file.Close()
	if err := h.service.ImportUsers(c.Request.Context(), file); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "imported"})
}
