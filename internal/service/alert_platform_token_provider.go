package service

import (
	"context"
	"strings"
	"time"

	"go-permission-system/internal/pkg/platformhttp"
)

type tokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type tokenProviderFunc func(ctx context.Context) (string, error)

func (f tokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

func (s *AlertService) cachedToken(ctx context.Context, cacheKey string, fetch func(ctx context.Context) (token string, expiresInSeconds int, err error)) (string, error) {
	if s.redis != nil {
		if v, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && strings.TrimSpace(v) != "" {
			return v, nil
		}
	}
	token, expiresIn, err := fetch(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return "", err
	}
	if s.redis != nil {
		ttl := time.Duration(expiresIn-120) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		_ = s.redis.Set(ctx, cacheKey, token, ttl).Err()
	}
	return token, nil
}

func (s *AlertService) platformHTTPClient() platformhttp.Client {
	return platformhttp.Client{Timeout: 5 * time.Second}
}
