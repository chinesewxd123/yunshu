package middleware

import (
	"runtime/debug"

	logx "go-permission-system/internal/pkg/logger"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error.Error("panic recovered",
					"panic", rec,
					"path", c.Request.URL.Path,
					"stack", string(debug.Stack()),
				)
				response.Error(c, apperror.Internal("internal server error"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
