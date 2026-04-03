package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegistrationRateLimit applies simple Redis-backed limits on registration attempts.
// - IP level: 5 attempts per minute, then temporary ban 30 minutes
// - Email/username level: 3 attempts per hour
func RegistrationRateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Check if IP is currently banned
		ip := c.ClientIP()
		banKey := fmt.Sprintf("ban:ip:%s", ip)
		if rdb.Exists(ctx, banKey).Val() > 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Your IP is temporarily blocked due to abuse."})
			return
		}

		// IP rate limit: 5 per minute
		ipKey := fmt.Sprintf("rl:register:ip:%s", ip)
		ipLimit := int64(5)
		if n, err := rdb.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				rdb.Expire(ctx, ipKey, time.Minute)
			}
			if n > ipLimit {
				// set a temporary ban
				rdb.Set(ctx, banKey, "1", 30*time.Minute)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many registration attempts from this IP, try again later."})
				return
			}
		}

		// Try to extract email/username from body (best-effort)
		var payload struct {
			Email    string `json:"email" form:"email"`
			Username string `json:"username" form:"username"`
		}
		_ = c.ShouldBind(&payload)
		if payload.Email != "" {
			emailKey := fmt.Sprintf("rl:register:email:%s", payload.Email)
			emailLimit := int64(3)
			if n, err := rdb.Incr(ctx, emailKey).Result(); err == nil {
				if n == 1 {
					rdb.Expire(ctx, emailKey, time.Hour)
				}
				if n > emailLimit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many registration attempts for this email, try later."})
					return
				}
			}
		}

		// Continue to handler
		c.Next()
	}
}
