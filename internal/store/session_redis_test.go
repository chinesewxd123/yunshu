package store

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestValidateAccessTokenSession_NilClient(t *testing.T) {
	err := ValidateAccessTokenSession(context.Background(), nil, "tid")
	if !errors.Is(err, ErrRedisRequired) {
		t.Fatalf("want ErrRedisRequired, got %v", err)
	}
}

func TestValidateAccessTokenSession_NotFound(t *testing.T) {
	// miniredis would be ideal; without it only test nil client / empty token
	err := ValidateAccessTokenSession(context.Background(), nil, "")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("want ErrSessionNotFound for empty token, got %v", err)
	}
	_ = redis.Nil
}
