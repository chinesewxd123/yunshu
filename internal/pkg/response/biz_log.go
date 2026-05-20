package response

import (
	"errors"
	"net/http"

	"yunshu/internal/pkg/apperror"
	logx "yunshu/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

const ctxKeyBizErrorLogged = "biz_error_logged"

// logHTTPError 记录 API 层返回的错误（每个请求只记一次，避免与中间件重复刷屏）。
func logHTTPError(c *gin.Context, err error) {
	if err == nil || c == nil || logx.Default() == nil {
		return
	}
	if apperror.AlreadyLogged(err) {
		return
	}
	if _, ok := c.Get(ctxKeyBizErrorLogged); ok {
		return
	}
	c.Set(ctxKeyBizErrorLogged, true)

	log := logx.Biz("http.api").WithLayer(logx.LayerAPI).W(c.Request.Context())
	attrs := []any{
		"method", c.Request.Method,
	}
	if route := c.FullPath(); route != "" {
		attrs = append(attrs, "path", route)
	} else if c.Request.URL != nil {
		attrs = append(attrs, "path", c.Request.URL.Path)
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs = append(attrs, "error_code", appErr.ErrorCode, "reason", appErr.Reason, "http_status", appErr.StatusCode)
		if appErr.StatusCode >= http.StatusInternalServerError {
			log.Errorw(err, "API request failed", attrs...)
			return
		}
		log.Warnw("API request rejected", append(attrs, "error", err.Error())...)
		return
	}
	log.Errorw(err, "API request failed", attrs...)
}
