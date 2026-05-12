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
			logger.Info.Warn("parse token failed", "error", err)
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
		c.Next()
	}
}
