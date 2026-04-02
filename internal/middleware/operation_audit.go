package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/auth"
	logx "go-permission-system/internal/pkg/logger"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	maxAuditBodyBytes  = 8192
	maxStoredBodyRunes = 4000
)

var sensitiveJSONKeyPattern = regexp.MustCompile(`"(?i)(password|code|token|authorization|secret)"\s*:\s*"[^"]*"`)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	_, _ = w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyCaptureWriter) WriteString(s string) (int, error) {
	_, _ = w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// OperationAudit 记录已鉴权请求的审计信息（参考 gin-vue-admin 操作日志）
func OperationAudit(opSvc *service.OperationLogService, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		reqBody := ""
		if shouldCaptureRequestBody(c) {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAuditBodyBytes))
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(raw))
				reqBody = maskSensitiveJSON(string(raw))
				reqBody = truncateRunes(reqBody, maxStoredBodyRunes)
			}
		}

		blw := &bodyCaptureWriter{ResponseWriter: c.Writer, buf: bytes.Buffer{}}
		c.Writer = blw

		start := time.Now()
		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		if status == 0 {
			status = http.StatusOK
		}

		respStr := maskSensitiveJSON(blw.buf.String())
		respStr = truncateRunes(respStr, maxStoredBodyRunes)

		// capture headers and mask sensitive values
		headersMap := map[string][]string{}
		for k, v := range c.Request.Header {
			headersMap[k] = append([]string(nil), v...)
		}
		maskHeaders(headersMap)
		headersJSON := ""
		if b, err := json.Marshal(headersMap); err == nil {
			headersJSON = truncateRunes(string(b), maxStoredBodyRunes)
		}

		entry := model.OperationLog{
			UserID:         user.ID,
			Username:       user.Username,
			Nickname:       user.Nickname,
			IP:             c.ClientIP(),
			RequestHeaders: headersJSON,
			Method:         c.Request.Method,
			Path:           path,
			StatusCode:     status,
			RequestBody:    reqBody,
			ResponseBody:   respStr,
			LatencyMs:      latency,
		}

		if err := opSvc.Record(c.Request.Context(), entry); err != nil && logger != nil {
			logger.Error.Error("operation audit persist failed", "error", err, "path", path)
		}
	}
}

func maskHeaders(h map[string][]string) {
	if h == nil {
		return
	}
	sensitive := map[string]struct{}{
		"Authorization": {},
		"Cookie":        {},
		"Set-Cookie":    {},
		"X-Auth-Token":  {},
		"Token":         {},
	}
	for k := range h {
		if _, ok := sensitive[k]; ok {
			h[k] = []string{"***"}
		}
	}
}

func shouldCaptureRequestBody(c *gin.Context) bool {
	method := c.Request.Method
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		return false
	}
	ct := c.ContentType()
	if strings.HasPrefix(ct, "multipart/") {
		return false
	}
	if cl := c.Request.ContentLength; cl > 0 && cl > maxAuditBodyBytes {
		return false
	}
	return true
}

func maskSensitiveJSON(s string) string {
	return sensitiveJSONKeyPattern.ReplaceAllString(s, `"$1":"***"`)
}

func truncateRunes(s string, max int) string {
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
