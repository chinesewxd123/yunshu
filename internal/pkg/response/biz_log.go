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

	attrs := []any{
		"layer", "api",
		"method", c.Request.Method,
	}
	if route := c.FullPath(); route != "" {
		attrs = append(attrs, "path", route)
	} else if c.Request.URL != nil {
		attrs = append(attrs, "path", c.Request.URL.Path)
	}
	if rid, ok := c.Get("request_id"); ok {
		attrs = append(attrs, "request_id", rid)
	}
	if uid, ok := c.Get("user_id"); ok {
		attrs = append(attrs, "user_id", uid)
	}

	log := logx.Default()
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		attrs = append(attrs, "error_code", appErr.ErrorCode, "http_status", appErr.StatusCode)
		if appErr.StatusCode >= http.StatusInternalServerError {
			log.Error.Error("api business error", append(attrs, "error", err.Error())...)
			return
		}
		log.Info.Debug("api client error", append(attrs, "error", err.Error())...)
		return
	}
	log.Error.Error("api business error", append(attrs, "error", err.Error())...)
}
