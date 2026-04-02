package service

import (
	"context"
	"errors"
	"io"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/repository"

	"github.com/casbin/casbin/v2"
	"github.com/xuri/excelize/v2"
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
	if err := s.ensureUserUnique(ctx, 0, req.Username, req.Email); err != nil {
		return nil, err
	}

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

	email := normalizeEmail(req.Email)
	user := model.User{
		Username: strings.TrimSpace(req.Username),
		Email:    &email,
		Password: hashedPassword,
		Nickname: strings.TrimSpace(req.Nickname),
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
		user.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		email := normalizeEmail(*req.Email)
		if err = s.ensureUserUnique(ctx, user.ID, user.Username, email); err != nil {
			return nil, err
		}
		user.Email = &email
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

func (s *UserService) ensureUserUnique(ctx context.Context, currentID uint, username, email string) error {
	if strings.TrimSpace(username) != "" {
		existing, err := s.userRepo.GetByUsername(ctx, strings.TrimSpace(username))
		if err == nil && existing.ID != currentID {
			return apperror.Conflict("username already exists")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if strings.TrimSpace(email) != "" {
		existing, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
		if err == nil && existing.ID != currentID {
			return apperror.Conflict("email already exists")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	return nil
}

// ListAll returns all users for export.
func (s *UserService) ListAll(ctx context.Context) ([]model.User, error) {
	return s.userRepo.ListAll(ctx)
}

// ImportUsers reads an Excel file from reader and creates users.
func (s *UserService) ImportUsers(ctx context.Context, r io.Reader) error {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}

	// Expect header row in first line: ID,Username,Nickname,Email,Status
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		username := strings.TrimSpace(row[1])
		if username == "" {
			continue
		}
		var nickname string
		var emailPtr *string
		var status int = int(model.StatusEnabled)
		if len(row) >= 3 {
			nickname = strings.TrimSpace(row[2])
		}
		if len(row) >= 4 {
			e := strings.TrimSpace(row[3])
			if e != "" {
				emailPtr = &e
			}
		}
		if len(row) >= 5 {
			// try parse status
			if strings.TrimSpace(row[4]) == "0" {
				status = int(model.StatusDisabled)
			}
		}

		// skip if user exists
		exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, username, "")
		if err == nil && exists {
			continue
		}

		hashed, _ := password.Hash("123456")
		user := model.User{
			Username: username,
			Nickname: nickname,
			Email:    emailPtr,
			Password: hashed,
			Status:   status,
		}
		_ = s.userRepo.Create(ctx, &user)
	}
	return nil
}
