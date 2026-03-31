package auth

import (
	"errors"
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
	ID        uint     `json:"id"`
	Username  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Status    int      `json:"status"`
	RoleCodes []string `json:"role_codes"`
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
