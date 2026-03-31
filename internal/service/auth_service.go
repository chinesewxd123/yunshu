package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-permission-system/internal/config"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/repository"
	"go-permission-system/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.UserRepository
	redis    *redis.Client
	cfg      config.AuthConfig
}

func NewAuthService(userRepo *repository.UserRepository, redis *redis.Client, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redis,
		cfg:      cfg,
	}
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Unauthorized("invalid username or password")
		}
		return nil, err
	}

	if user.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("user is disabled")
	}
	if err = password.Compare(user.Password, req.Password); err != nil {
		return nil, apperror.Unauthorized("invalid username or password")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute)
	tokenID := uuid.NewString()
	token, err := auth.GenerateToken(s.cfg.JWTSecret, auth.Claims{
		UserID:   user.ID,
		Username: user.Username,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	})
	if err != nil {
		return nil, err
	}

	if s.redis != nil {
		key := store.AccessTokenKey(tokenID)
		if err = s.redis.Set(ctx, key, user.ID, time.Until(expiresAt)).Err(); err != nil {
			return nil, err
		}
	}

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      NewUserDetailResponse(*user),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenID string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, store.AccessTokenKey(tokenID)).Err()
}

func (s *AuthService) Me(ctx context.Context, userID uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}
