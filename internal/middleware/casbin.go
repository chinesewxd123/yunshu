package middleware

import (
	"log/slog"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func Authorize(enforcer *casbin.SyncedEnforcer, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("未登录"))
			c.Abort()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		allowed, err := enforcer.Enforce(service.UserSubject(user.ID), path, c.Request.Method)
		if err != nil {
			logger.Error("casbin authorize failed", "error", err, "path", path, "method", c.Request.Method)
			response.Error(c, apperror.Internal("权限校验失败"))
			c.Abort()
			return
		}
		if !allowed {
			response.Error(c, apperror.Forbidden("无访问权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}
