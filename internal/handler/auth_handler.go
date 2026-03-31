package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login godoc
// @Summary Login
// @Description Login with username and password and return a JWT access token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login request"
// @Success 200 {object} response.Body{data=service.LoginResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Logout godoc
// @Summary Logout
// @Description Logout the current user by invalidating the current token.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("unauthorized"))
		return
	}

	if err := h.service.Logout(c.Request.Context(), claims.TokenID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "logout success"})
}

// Me godoc
// @Summary Get current user
// @Description Get the profile of the current authenticated user.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("unauthorized"))
		return
	}

	data, err := h.service.Me(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}
