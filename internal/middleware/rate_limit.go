package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegistrationRateLimit applies simple Redis-backed limits on registration attempts.
// - IP level: 20 attempts per minute, then temporary ban 30 minutes
// - Email/username level: 3 attempts per hour
func RegistrationRateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		ctx := context.Background()

		// Check if IP is currently banned
		ip := c.ClientIP()
		banKey := fmt.Sprintf("ban:ip:%s", ip)
		if rdb.Exists(ctx, banKey).Val() > 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "当前 IP 因频繁请求已被临时封禁，请稍后再试"})
			return
		}

		// IP rate limit: 20 per minute
		ipKey := fmt.Sprintf("rl:register:ip:%s", ip)
		ipLimit := int64(20)
		if n, err := rdb.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				rdb.Expire(ctx, ipKey, time.Minute)
			}
			if n > ipLimit {
				// set a temporary ban
				rdb.Set(ctx, banKey, "1", 30*time.Minute)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "当前 IP 注册尝试过于频繁，请稍后再试"})
				return
			}
		}

		// Try to extract email/username from body (best-effort).
		// Read and then restore body so downstream handlers can still bind it.
		var payload struct {
			Email    string `json:"email" form:"email"`
			Username string `json:"username" form:"username"`
		}
		if c.Request != nil && c.Request.Body != nil {
			if bodyBytes, err := io.ReadAll(c.Request.Body); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				_ = json.Unmarshal(bodyBytes, &payload)
				// Ensure subsequent binders can still read the same payload
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}
		if payload.Email != "" {
			emailKey := fmt.Sprintf("rl:register:email:%s", strings.ToLower(strings.TrimSpace(payload.Email)))
			emailLimit := int64(3)
			if n, err := rdb.Incr(ctx, emailKey).Result(); err == nil {
				if n == 1 {
					rdb.Expire(ctx, emailKey, time.Hour)
				}
				if n > emailLimit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "该邮箱注册尝试次数过多，请稍后再试"})
					return
				}
			}
		}

		// Continue to handler
		c.Next()
	}
}
