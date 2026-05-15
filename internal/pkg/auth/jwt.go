package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ContextClaimsKey = "auth_claims"
	ContextUserKey   = "auth_current_user"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	TokenID  string `json:"token_id"`
	jwt.RegisteredClaims
}

type CurrentUser struct {
	ID           uint     `json:"id"`
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	Status       int      `json:"status"`
	DepartmentID *uint    `json:"department_id,omitempty"`
	RoleCodes    []string `json:"role_codes"`
	GroupCodes   []string `json:"group_codes"`
}

// IsSuperAdminRole reports whether the subject has the built-in super-admin role.
func IsSuperAdminRole(roleCodes []string) bool {
	for _, code := range roleCodes {
		if strings.TrimSpace(code) == "super-admin" {
			return true
		}
	}
	return false
}

// CanManageOtherUsersLoginPassword reports whether the subject may set another user's login password in user management.
// 与内置超级管理员角色对齐；普通用户不可通过用户更新接口写入 password 字段。
func CanManageOtherUsersLoginPassword(roleCodes []string) bool {
	return IsSuperAdminRole(roleCodes)
}

func GenerateToken(secret string, claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}
	return claims, nil
}

type requestUserCtxKey struct{}

// RequestContext 将当前登录用户写入标准 context，供 service 层做租户/项目等校验（无登录态时等价于 c.Request.Context()）。
func RequestContext(c *gin.Context) context.Context {
	if c == nil {
		return context.Background()
	}
	base := c.Request.Context()
	if u, ok := CurrentUserFromContext(c); ok && u != nil {
		return context.WithValue(base, requestUserCtxKey{}, u)
	}
	return base
}

// RequestUserFromContext 读取由 RequestContext 注入的当前用户；不存在则 ok=false。
func RequestUserFromContext(ctx context.Context) (*CurrentUser, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(requestUserCtxKey{})
	if v == nil {
		return nil, false
	}
	u, ok := v.(*CurrentUser)
	return u, ok
}

func CurrentUserFromContext(c *gin.Context) (*CurrentUser, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}
	user, ok := value.(*CurrentUser)
	return user, ok
}

func ClaimsFromContext(c *gin.Context) (*Claims, bool) {
	value, exists := c.Get(ContextClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*Claims)
	return claims, ok
}
