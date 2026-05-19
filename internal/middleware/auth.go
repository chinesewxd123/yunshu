package middleware

import (
	"errors"
	"strings"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	"yunshu/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9" // redis.Client 由路由注入
)

func respondSessionStoreError(c *gin.Context, _ *logx.Logger, err error) {
	switch {
	case errors.Is(err, store.ErrSessionNotFound):
		response.Error(c, constants.ErrLoginSessionExpired)
	case errors.Is(err, store.ErrRedisRequired), errors.Is(err, store.ErrRedisUnavailable):
		httpLog("http.auth").Error("redis session validation failed", "error", err)
		response.Error(c, constants.ErrInternal)
	default:
		httpLog("http.auth").Error("session validation failed", "error", err)
	}
}

func Auth(secret string, redisClient *redis.Client, userRepo *repository.UserRepository, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, constants.ErrMissingAuthHeader)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(secret, tokenString)
		if err != nil {
			httpLog("http.auth").Warn("parse token failed", "error", err)
			response.Error(c, constants.ErrAccessTokenInvalid)
			c.Abort()
			return
		}

		// 会话白名单：redis.Nil=已过期；Redis 故障=500，禁止无 Redis 时放行
		if err = store.ValidateAccessTokenSession(c.Request.Context(), redisClient, claims.TokenID); err != nil {
			respondSessionStoreError(c, logger, err)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Error(c, constants.ErrAccountPrincipalNotFound)
			c.Abort()
			return
		}
		if user.Status != model.StatusEnabled {
			response.Error(c, constants.ErrAccountDisabled)
			c.Abort()
			return
		}

		groupCodes := make([]string, 0, len(user.Groups))
		for _, g := range user.Groups {
			if c := strings.TrimSpace(g.Code); c != "" {
				groupCodes = append(groupCodes, c)
			}
		}
		currentUser := &auth.CurrentUser{
			ID:           user.ID,
			Username:     user.Username,
			Nickname:     user.Nickname,
			Status:       user.Status,
			DepartmentID: user.DepartmentID,
			RoleCodes:    model.ExtractRoleCodes(user.Roles),
			GroupCodes:   groupCodes,
		}

		c.Set(auth.ContextClaimsKey, claims)
		c.Set(auth.ContextUserKey, currentUser)
		c.Request = c.Request.WithContext(logx.WithUser(c.Request.Context(), user.ID, user.Username))
		c.Next()
	}
}
