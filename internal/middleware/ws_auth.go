package middleware

import (
	"strings"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"
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
			response.Error(c, constants.ErrWSMissingTokenParam)
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(secret, tokenString)
		if err != nil {
			if logger != nil {
				logger.Info.Warn("parse ws token failed", "error", err)
			}
			response.Error(c, constants.ErrAccessTokenInvalid)
			c.Abort()
			return
		}

		if redisClient != nil {
			if _, err = redisClient.Get(c.Request.Context(), store.AccessTokenKey(claims.TokenID)).Result(); err != nil {
				response.Error(c, constants.ErrLoginSessionExpired)
				c.Abort()
				return
			}
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
