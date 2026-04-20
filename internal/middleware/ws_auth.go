package middleware

import (
	"strings"

	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	"yunshu/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// WSAuth authenticates websocket handshake requests.
// Browsers can't set custom headers for WebSocket, so we accept token from query:
// - token=<jwt>  or access_token=<jwt>
func WSAuth(secret string, redisClient *redis.Client, userRepo *repository.UserRepository, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := strings.TrimSpace(c.Query("token"))
		if tokenString == "" {
			tokenString = strings.TrimSpace(c.Query("access_token"))
		}
		if tokenString == "" {
			response.Error(c, apperror.Unauthorized("缺少 token 参数"))
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(secret, tokenString)
		if err != nil {
			if logger != nil {
				logger.Info.Warn("parse ws token failed", "error", err)
			}
			response.Error(c, apperror.Unauthorized("Token 无效"))
			c.Abort()
			return
		}

		if redisClient != nil {
			if _, err = redisClient.Get(c.Request.Context(), store.AccessTokenKey(claims.TokenID)).Result(); err != nil {
				response.Error(c, apperror.Unauthorized("登录已失效"))
				c.Abort()
				return
			}
		}

		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Error(c, apperror.Unauthorized("用户不存在"))
			c.Abort()
			return
		}
		if user.Status != model.StatusEnabled {
			response.Error(c, apperror.Forbidden("用户已被禁用"))
			c.Abort()
			return
		}

		currentUser := &auth.CurrentUser{
			ID:        user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Status:    user.Status,
			RoleCodes: model.ExtractRoleCodes(user.Roles),
		}

		c.Set(auth.ContextClaimsKey, claims)
		c.Set(auth.ContextUserKey, currentUser)
		c.Next()
	}
}

