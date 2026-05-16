package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrSessionNotFound 表示 access token 在 Redis 中不存在（已登出或过期）。
	ErrSessionNotFound = errors.New("access token session not found")
	// ErrRedisRequired 表示会话校验依赖 Redis，但客户端未配置。
	ErrRedisRequired = errors.New("redis client is required for session validation")
	// ErrRedisUnavailable 表示 Redis 网络/服务异常（非 key 不存在）。
	ErrRedisUnavailable = errors.New("redis unavailable")
)

// ValidateAccessTokenSession 校验 JWT 对应会话是否仍在 Redis 白名单中。
// - 成功：返回 nil
// - redis.Nil：ErrSessionNotFound（应对应 401/会话过期，而非 500）
// - 其它 Redis 错误：ErrRedisUnavailable（应对应 500，禁止降级放行）
func ValidateAccessTokenSession(ctx context.Context, rdb *redis.Client, tokenID string) error {
	if rdb == nil {
		return ErrRedisRequired
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return ErrSessionNotFound
	}
	_, err := rdb.Get(ctx, AccessTokenKey(tokenID)).Result()
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return ErrSessionNotFound
	}
	return fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
}
