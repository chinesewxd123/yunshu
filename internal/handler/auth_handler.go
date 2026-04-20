package handler

import (
	"context"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service   *service.AuthService
	loginLogs *service.LoginLogService
}

// NewAuthHandler 创建相关逻辑。
func NewAuthHandler(service *service.AuthService, loginLogs *service.LoginLogService) *AuthHandler {
	return &AuthHandler{service: service, loginLogs: loginLogs}
}

func loginErrMessage(err error) string {
	if appErr, ok := apperror.IsAppError(err); ok {
		return appErr.Message
	}
	return "登录失败"
}

func (h *AuthHandler) recordLogin(c *gin.Context, username, source string, success bool, detail string, userID *uint) {
	if h.loginLogs == nil || username == "" {
		return
	}
	status := model.LoginLogStatusFail
	if success {
		status = model.LoginLogStatusSuccess
	}
	entry := model.LoginLog{
		Username:  username,
		IP:        c.ClientIP(),
		Status:    status,
		Detail:    detail,
		UserAgent: c.GetHeader("User-Agent"),
		Source:    source,
		UserID:    userID,
	}
	_ = h.loginLogs.Record(c.Request.Context(), entry)
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
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 409 {object} response.Body "email already registered"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/verification-code [post]
func (h *AuthHandler) SendEmailCode(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.SendEmailCodeRequest) (*service.SendEmailCodeResponse, error) {
		return h.service.SendEmailCodeWithIP(ctx, service.SendEmailCodeWithIPRequest{
			SendEmailCodeRequest: req,
			ClientIP:             c.ClientIP(),
		})
	})
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
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/login-code [post]
func (h *AuthHandler) SendLoginCodeByUsername(c *gin.Context) {
	handleJSON(c, h.service.SendLoginCodeByUsername)
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
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 409 {object} response.Body "cooldown in effect"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/password-login-code [post]
func (h *AuthHandler) SendPasswordLoginCode(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.SendPasswordLoginCodeRequest) (*service.SendPasswordLoginCodeResponse, error) {
		return h.service.SendPasswordLoginCode(ctx, req.Username)
	})
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
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.LoginRequest) (*service.LoginResponse, error) {
		data, err := h.service.Login(ctx, req)
		if err != nil {
			h.recordLogin(c, req.Username, model.LoginSourcePassword, false, loginErrMessage(err), nil)
			return nil, err
		}
		uid := data.User.ID
		h.recordLogin(c, data.User.Username, model.LoginSourcePassword, true, "登录成功", &uid)
		return data, nil
	})
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
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/email-login [post]
func (h *AuthHandler) EmailLogin(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.EmailLoginRequest) (*service.LoginResponse, error) {
		data, err := h.service.EmailLogin(ctx, req)
		if err != nil {
			h.recordLogin(c, req.Email, model.LoginSourceEmail, false, loginErrMessage(err), nil)
			return nil, err
		}
		uid := data.User.ID
		h.recordLogin(c, data.User.Username, model.LoginSourceEmail, true, "登录成功", &uid)
		return data, nil
	})
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
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	handleJSONCreated(c, h.service.Register)
}

// Logout godoc
// @Summary Logout
// @Description Logout the current user by invalidating the current token.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}

	if err := h.service.Logout(c.Request.Context(), claims.TokenID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "退出登录成功"})
}

// Me godoc
// @Summary Get current user
// @Description Get the profile of the current authenticated user.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}

	data, err := h.service.Me(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// UpdateProfile godoc
// @Summary 更新当前用户资料
// @Description 更新当前登录用户的昵称和邮箱。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.UpdateProfileRequest true "更新资料请求"
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 409 {object} response.Body "邮箱已存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/me [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleJSON(c, func(ctx context.Context, req service.UpdateProfileRequest) (*service.UserDetailResponse, error) {
		return h.service.UpdateProfile(ctx, user.ID, req)
	})
}

// ChangePassword godoc
// @Summary 修改当前用户密码
// @Description 使用旧密码校验后修改当前登录用户密码。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录或登录已失效"))
		return
	}
	handleJSONOK(c, gin.H{"message": "密码修改成功"}, func(ctx context.Context, req service.ChangePasswordRequest) error {
		return h.service.ChangePassword(ctx, user.ID, req)
	})
}
