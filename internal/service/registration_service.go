package service

import (
	"context"
	"errors"
	"strings"

	"go-permission-system/internal/config"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/mailer"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/repository"
	"go-permission-system/internal/store"

	"github.com/redis/go-redis/v9"
)

type RegistrationService struct {
	regRepo  *repository.RegistrationRequestRepository
	userRepo *repository.UserRepository
	redis    *redis.Client
	authCfg  config.AuthConfig
	mailer   mailer.Sender
	appName  string
}

func NewRegistrationService(
	regRepo *repository.RegistrationRequestRepository,
	userRepo *repository.UserRepository,
	redis *redis.Client,
	authCfg config.AuthConfig,
	mailer mailer.Sender,
	appName string,
) *RegistrationService {
	return &RegistrationService{
		regRepo:  regRepo,
		userRepo: userRepo,
		redis:    redis,
		authCfg:  authCfg,
		mailer:   mailer,
		appName:  appName,
	}
}

type ApplyRegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Nickname string `json:"nickname" binding:"required,max=128"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
}

type ReviewRequest struct {
	Status  int    `json:"status" binding:"required,oneof=1 2"`
	Comment string `json:"comment"`
}

func (s *RegistrationService) Apply(ctx context.Context, req ApplyRegisterRequest) error {
	email := normalizeEmail(req.Email)
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)

	if err := s.ensureRegistrationDoesNotExist(ctx, username, email); err != nil {
		return err
	}
	if err := s.validateEmailCode(ctx, emailCodeSceneRegister, email, req.Code); err != nil {
		return err
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return err
	}

	regReq := model.RegistrationRequest{
		Username: username,
		Email:    email,
		Nickname: nickname,
		Password: hashedPassword,
		Status:   model.RegistrationPending,
	}

	if err = s.regRepo.Create(ctx, &regReq); err != nil {
		return err
	}

	s.clearEmailCode(ctx, emailCodeSceneRegister, email)
	return nil
}

func (s *RegistrationService) List(ctx context.Context, keyword string, status *int, page, pageSize int) ([]model.RegistrationRequest, int64, error) {
	return s.regRepo.List(ctx, repository.RegistrationRequestListParams{
		Keyword:  keyword,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
}

func (s *RegistrationService) Review(ctx context.Context, id uint, reviewerID uint, req ReviewRequest) error {
	regReq, err := s.regRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New("registration request not found")
	}
	if regReq.Status != model.RegistrationPending {
		return errors.New("this request has already been reviewed")
	}

	newStatus := model.RegistrationRequestStatus(req.Status)
	err = s.regRepo.UpdateStatus(ctx, id, newStatus, reviewerID, req.Comment)
	if err != nil {
		return err
	}

	if newStatus == model.RegistrationApproved {
		user := model.User{
			Username: regReq.Username,
			Email:    &regReq.Email,
			Password: regReq.Password,
			Nickname: regReq.Nickname,
			Status:   model.StatusEnabled,
			Roles:    []model.Role{},
		}
		if err := s.userRepo.Create(ctx, &user); err != nil {
			return err
		}
	}

	return nil
}

func (s *RegistrationService) ensureRegistrationDoesNotExist(ctx context.Context, username, email string) error {
	count, err := s.regRepo.CountPending(ctx, username, email)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("a pending registration already exists for this username or email")
	}

	exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, username, email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("username or email already registered")
	}
	return nil
}

func (s *RegistrationService) validateEmailCode(ctx context.Context, scene, email, code string) error {
	if s.redis == nil {
		return nil
	}
	key := store.EmailCodeKey(scene, email)
	val, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return errors.New("verification code has expired or not sent")
	}
	if val != code {
		return errors.New("verification code is incorrect")
	}
	return nil
}

func (s *RegistrationService) clearEmailCode(ctx context.Context, scene, email string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(ctx, store.EmailCodeKey(scene, email))
}
