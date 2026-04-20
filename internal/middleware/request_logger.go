package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var requestSensitiveJSONKeyPattern = regexp.MustCompile(`"(?i)(password|code|token|authorization|secret|private_key|passphrase)"\s*:\s*"[^"]*"`)

// queryKeysToMask 访问日志 query 中可能携带 JWT 或密钥，避免落盘明文。
var queryKeysToMask = []string{
	"token", "access_token", "refresh_token", "password", "secret", "authorization",
}

func RequestLogger(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)

		reqBody := ""
		if shouldCaptureRequestLogBody(c) && c.Request != nil && c.Request.Body != nil {
			if bodyBytes, err := io.ReadAll(c.Request.Body); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				reqBody = truncateRequestLogRunes(maskRequestLogSensitiveJSON(string(bodyBytes)), 1000)
			}
		}

		c.Next()
		latency := time.Since(start)
		route := c.FullPath()
		if strings.TrimSpace(route) == "" {
			route = c.Request.URL.Path
		}

		attrs := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"path", route,
			"query", maskSensitiveQuery(c.Request.URL.RawQuery),
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if reqBody != "" {
			attrs = append(attrs, "request_body", reqBody)
		}
		if user, ok := auth.CurrentUserFromContext(c); ok {
			attrs = append(attrs, "user_id", user.ID, "username", user.Username)
		}
		if errorCode, ok := c.Get("error_code"); ok {
			attrs = append(attrs, "error_code", errorCode)
		}
		if errorMessage, ok := c.Get("error_message"); ok {
			attrs = append(attrs, "error_message", errorMessage)
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

func shouldCaptureRequestLogBody(c *gin.Context) bool {
	method := c.Request.Method
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
		return false
	}
	ct := c.ContentType()
	if strings.HasPrefix(ct, "multipart/") {
		return false
	}
	return true
}

func maskRequestLogSensitiveJSON(s string) string {
	return requestSensitiveJSONKeyPattern.ReplaceAllString(s, `"$1":"***"`)
}

func maskSensitiveQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return "[query_parse_error]"
	}
	for key := range q {
		for _, mk := range queryKeysToMask {
			if strings.EqualFold(key, mk) {
				q.Set(key, "***")
				break
			}
		}
	}
	return q.Encode()
}

func truncateRequestLogRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…(truncated)"
}
