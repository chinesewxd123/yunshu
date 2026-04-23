package middleware

import (
	"runtime/debug"

	"yunshu/internal/pkg/apperror"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"

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
				response.Error(c, apperror.Internal("服务器内部错误"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
