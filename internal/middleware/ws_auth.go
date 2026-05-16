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
			ID:         user.ID,
			Username:   user.Username,
			Nickname:   user.Nickname,
			Status:     user.Status,
			RoleCodes:  model.ExtractRoleCodes(user.Roles),
			GroupCodes: groupCodes,
		}

		c.Set(auth.ContextClaimsKey, claims)
		c.Set(auth.ContextUserKey, currentUser)
		c.Next()
	}
}
