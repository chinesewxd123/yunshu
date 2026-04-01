package middleware

import (
	"time"

	logx "go-permission-system/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

func RequestLogger(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"client_ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		if c.Writer.Status() >= 500 {
			logger.Error.Error("http request completed", attrs...)
			return
		}
		if c.Writer.Status() >= 400 {
			logger.Info.Warn("http request completed", attrs...)
			return
		}
		logger.Info.Info("http request completed", attrs...)
	}
}
