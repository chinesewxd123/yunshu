package middleware

import (
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
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
			response.Error(c, apperror.Unauthorized("缺少或非法授权请求头"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseToken(secret, tokenString)
		if err != nil {
			logger.Info.Warn("parse token failed", "error", err)
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
			ID:           user.ID,
			Username:     user.Username,
			Nickname:     user.Nickname,
			Status:       user.Status,
			DepartmentID: user.DepartmentID,
			RoleCodes:    model.ExtractRoleCodes(user.Roles),
		}

		c.Set(auth.ContextClaimsKey, claims)
		c.Set(auth.ContextUserKey, currentUser)
		c.Next()
	}
}
