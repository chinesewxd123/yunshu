package service

import (
	"context"
	"errors"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
	enforcer *casbin.SyncedEnforcer
}

func NewUserService(userRepo *repository.UserRepository, roleRepo *repository.RoleRepository, enforcer *casbin.SyncedEnforcer) *UserService {
	return &UserService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		enforcer: enforcer,
	}
}

func (s *UserService) Create(ctx context.Context, req UserCreateRequest) (*UserDetailResponse, error) {
	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, apperror.BadRequest("some roles do not exist")
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	user := model.User{
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Status:   status,
		Roles:    roles,
	}
	if err = s.userRepo.Create(ctx, &user); err != nil {
		return nil, err
	}
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, err
	}

	response := NewUserDetailResponse(user)
	return &response, nil
}

func (s *UserService) Update(ctx context.Context, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Password != nil && *req.Password != "" {
		user.Password, err = password.Hash(*req.Password)
		if err != nil {
			return nil, err
		}
	}

	if err = s.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) Delete(ctx context.Context, id uint) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("user not found")
		}
		return err
	}

	if err = s.userRepo.Delete(ctx, user); err != nil {
		return err
	}
	_, err = s.enforcer.DeleteUser(UserSubject(id))
	return err
}

func (s *UserService) Detail(ctx context.Context, id uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) List(ctx context.Context, query UserListQuery) (*pagination.Result[UserDetailResponse], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	users, total, err := s.userRepo.List(ctx, repository.UserListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	list := make([]UserDetailResponse, 0, len(users))
	for _, user := range users {
		list = append(list, NewUserDetailResponse(user))
	}

	return &pagination.Result[UserDetailResponse]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *UserService) AssignRoles(ctx context.Context, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, apperror.BadRequest("some roles do not exist")
	}

	if err = s.userRepo.ReplaceRoles(ctx, user, roles); err != nil {
		return nil, err
	}

	user.Roles = roles
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, err
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}
