package middleware

import (
	"errors"
	"runtime/debug"
	"yunshu/internal/pkg/constants"

	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/svclog"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				svclog.HTTP("http.recovery").Errorw(errors.New("panic"), "Recovered HTTP panic",
					"panic", rec,
					"path", c.Request.URL.Path,
					"stack", string(debug.Stack()),
				)
				response.Error(c, constants.ErrInternal)
				c.Abort()
			}
		}()
		c.Next()
	}
}
