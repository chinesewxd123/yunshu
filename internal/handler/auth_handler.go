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

// SendEmailCode godoc
// @Summary Send verification code
// @Description Send an email verification code for login or registration and cache it in Redis.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendEmailCodeRequest true "Send email code request"
// @Success 200 {object} response.Body{data=service.SendEmailCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "user not found"
// @Failure 409 {object} response.Body "email already registered"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/verification-code [post]
func (h *AuthHandler) SendEmailCode(c *gin.Context) {
	var req service.SendEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.SendEmailCode(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// SendLoginCodeByUsername godoc
// @Summary Send login verification code by username
// @Description Send an email verification code to the user's registered email address for login.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendLoginCodeByUsernameRequest true "Send login code by username request"
// @Success 200 {object} response.Body{data=service.SendEmailCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/login-code [post]
func (h *AuthHandler) SendLoginCodeByUsername(c *gin.Context) {
	var req service.SendLoginCodeByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.SendLoginCodeByUsername(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// SendPasswordLoginCode godoc
// @Summary Send password login verification code
// @Description Generate a 6-digit verification code for password login and cache it in Redis. The code is returned in the response for demonstration purposes in development. In production, it would be sent via email or SMS.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendPasswordLoginCodeRequest true "Send password login code request"
// @Success 200 {object} response.Body{data=service.SendPasswordLoginCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "user not found"
// @Failure 409 {object} response.Body "cooldown in effect"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/password-login-code [post]
func (h *AuthHandler) SendPasswordLoginCode(c *gin.Context) {
	var req service.SendPasswordLoginCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.SendPasswordLoginCode(c.Request.Context(), req.Username)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Login godoc
// @Summary Login with password
// @Description Login with username, password and verification code. This is kept as a fallback entry for administrators.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Password login request"
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

// EmailLogin godoc
// @Summary Login with email verification code
// @Description Login with email and a 6-digit verification code delivered through SMTP and cached in Redis.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.EmailLoginRequest true "Email verification login request"
// @Success 200 {object} response.Body{data=service.LoginResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "user not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/email-login [post]
func (h *AuthHandler) EmailLogin(c *gin.Context) {
	var req service.EmailLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.EmailLogin(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Register godoc
// @Summary Register with email verification code
// @Description Register a new user by submitting username, email, password and the verification code sent by email.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Email registration request"
// @Success 201 {object} response.Body{data=service.RegisterResponse} "created"
// @Failure 400 {object} response.Body "bad request"
// @Failure 409 {object} response.Body "conflict"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	data, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, data)
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
