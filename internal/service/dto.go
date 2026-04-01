package service

import (
	"time"

	"go-permission-system/internal/model"
)

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	CaptchaKey string `json:"captcha_key" binding:"required"`
	Code       string `json:"code" binding:"required,len=4,numeric"`
}

type SendEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email,max=128"`
	Scene string `json:"scene" binding:"required,oneof=login register"`
}

type SendLoginCodeByUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}
type SendPasswordLoginCodeRequest struct {
	Username string `json:"username" binding:"required"`
}

type SendPasswordLoginCodeResponse struct {
	CaptchaKey string `json:"captcha_key"`
	Image      string `json:"image"`
	ExpiresIn  int    `json:"expires_in"`
	CooldownIn int    `json:"cooldown_in"`
}
type SendEmailCodeResponse struct {
	Email      string `json:"email"`
	Scene      string `json:"scene"`
	ExpiresIn  int    `json:"expires_in"`
	CooldownIn int    `json:"cooldown_in"`
}

type EmailLoginRequest struct {
	Email string `json:"email" binding:"required,email,max=128"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Nickname string `json:"nickname" binding:"required,max=128"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
}

type RegisterResponse struct {
	Message string             `json:"message"`
	User    UserDetailResponse `json:"user"`
}

type LoginResponse struct {
	Token     string             `json:"token"`
	ExpiresAt time.Time          `json:"expires_at"`
	User      UserDetailResponse `json:"user"`
}

type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname" binding:"required,max=128"`
	Status   int    `json:"status"`
	RoleIDs  []uint `json:"role_ids"`
}

type UserUpdateRequest struct {
	Email    *string `json:"email" binding:"omitempty,email,max=128"`
	Nickname *string `json:"nickname" binding:"omitempty,max=128"`
	Password *string `json:"password" binding:"omitempty,min=6,max=64"`
	Status   *int    `json:"status"`
}

type UserAssignRolesRequest struct {
	RoleIDs []uint `json:"role_ids"`
}

type UserListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type RoleCreateRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Code        string `json:"code" binding:"required,max=64"`
	Description string `json:"description" binding:"omitempty,max=255"`
	Status      int    `json:"status"`
}

type RoleUpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=64"`
	Code        *string `json:"code" binding:"omitempty,max=64"`
	Description *string `json:"description" binding:"omitempty,max=255"`
	Status      *int    `json:"status"`
}

type RoleListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type PermissionCreateRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Resource    string `json:"resource" binding:"required,max=191"`
	Action      string `json:"action" binding:"required,max=32"`
	Description string `json:"description" binding:"omitempty,max=255"`
}

type PermissionUpdateRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=64"`
	Resource    *string `json:"resource" binding:"omitempty,max=191"`
	Action      *string `json:"action" binding:"omitempty,max=32"`
	Description *string `json:"description" binding:"omitempty,max=255"`
}

type PermissionListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type PolicyGrantRequest struct {
	RoleID       uint `json:"role_id" binding:"required"`
	PermissionID uint `json:"permission_id" binding:"required"`
}

type RoleItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      int    `json:"status"`
}

type PermissionItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type UserDetailResponse struct {
	ID        uint       `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Nickname  string     `json:"nickname"`
	Status    int        `json:"status"`
	Roles     []RoleItem `json:"roles"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type PolicyItemResponse struct {
	RoleID         uint   `json:"role_id"`
	RoleName       string `json:"role_name"`
	RoleCode       string `json:"role_code"`
	PermissionID   uint   `json:"permission_id"`
	PermissionName string `json:"permission_name"`
	Resource       string `json:"resource"`
	Action         string `json:"action"`
}

func NewRoleItem(role model.Role) RoleItem {
	return RoleItem{
		ID:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		Status:      role.Status,
	}
}

func NewPermissionItem(permission model.Permission) PermissionItem {
	return PermissionItem{
		ID:          permission.ID,
		Name:        permission.Name,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Description: permission.Description,
	}
}

func NewUserDetailResponse(user model.User) UserDetailResponse {
	roles := make([]RoleItem, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, NewRoleItem(role))
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	return UserDetailResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     email,
		Nickname:  user.Nickname,
		Status:    user.Status,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
