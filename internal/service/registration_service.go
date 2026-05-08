package service

import (
	"context"
	"errors"
	"strings"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"
	"yunshu/internal/store"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RegistrationService struct {
	regRepo  *repository.RegistrationRequestRepository
	userRepo *repository.UserRepository
	redis    *redis.Client
	authCfg  config.AuthConfig
	mailer   mailer.Sender
	appName  string
}

// NewRegistrationService 创建相关逻辑。
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

// Apply 提交申请相关的业务逻辑。
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

// List 查询列表相关的业务逻辑。
func (s *RegistrationService) List(ctx context.Context, keyword string, status *int, page, pageSize int) ([]model.RegistrationRequest, int64, error) {
	return s.regRepo.List(ctx, repository.RegistrationRequestListParams{
		Keyword:  keyword,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
}

// Review 执行对应的业务逻辑。
func (s *RegistrationService) Review(ctx context.Context, id uint, reviewerID uint, req ReviewRequest) error {
	regReq, err := s.regRepo.GetByID(ctx, id)
	if err != nil {
		return constants.ErrRegistrationRequestNotFound
	}
	if regReq.Status != model.RegistrationPending {
		return constants.ErrRegistrationAlreadyProcessed
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
		return constants.ErrRegistrationDuplicatePending
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return constants.ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return constants.ErrEmailAlreadyRegistered
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
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
		if errors.Is(err, redis.Nil) {
			return constants.ErrCaptchaExpired
		}
		return err
	}
	if val != code {
		return constants.ErrCaptchaIncorrect
	}
	return nil
}

func (s *RegistrationService) clearEmailCode(ctx context.Context, scene, email string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(ctx, store.EmailCodeKey(scene, email))
}
