package service

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/repository"

	"github.com/casbin/casbin/v2"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo       *repository.UserRepository
	roleRepo       *repository.RoleRepository
	departmentRepo *repository.DepartmentRepository
	enforcer       *casbin.SyncedEnforcer
}

// NewUserService 创建相关逻辑。
func NewUserService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	departmentRepo *repository.DepartmentRepository,
	enforcer *casbin.SyncedEnforcer,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		departmentRepo: departmentRepo,
		enforcer:       enforcer,
	}
}

func isSuperAdmin(roleCodes []string) bool {
	for _, code := range roleCodes {
		if strings.TrimSpace(code) == "super-admin" {
			return true
		}
	}
	return false
}

func (s *UserService) accessibleDepartmentIDs(ctx context.Context, actor *auth.CurrentUser) ([]uint, error) {
	if actor == nil || actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, nil
	}
	return s.departmentRepo.ListDescendantIDsAndSelf(ctx, *actor.DepartmentID)
}

func (s *UserService) canAccessUser(ctx context.Context, actor *auth.CurrentUser, target *model.User) (bool, error) {
	if actor == nil || target == nil {
		return false, nil
	}
	if isSuperAdmin(actor.RoleCodes) || actor.ID == target.ID {
		return true, nil
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return false, nil
	}
	if target.DepartmentID == nil || *target.DepartmentID == 0 {
		return false, nil
	}
	ids, err := s.accessibleDepartmentIDs(ctx, actor)
	if err != nil {
		return false, err
	}
	return slices.Contains(ids, *target.DepartmentID), nil
}

// Create 创建相关的业务逻辑。
func (s *UserService) Create(ctx context.Context, req UserCreateRequest) (*UserDetailResponse, error) {
	if err := s.ensureUserUnique(ctx, 0, req.Username, req.Email); err != nil {
		return nil, err
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, apperror.BadRequest("部分角色不存在")
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
	var departmentID *uint
	if req.DepartmentID != nil && *req.DepartmentID > 0 {
		if _, err = s.departmentRepo.GetByID(ctx, *req.DepartmentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.BadRequest("所属部门不存在")
			}
			return nil, err
		}
		departmentID = req.DepartmentID
	}
	user := model.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        &email,
		Password:     hashedPassword,
		Nickname:     strings.TrimSpace(req.Nickname),
		Status:       status,
		DepartmentID: departmentID,
		Roles:        roles,
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

func (s *UserService) CreateByActor(ctx context.Context, actor *auth.CurrentUser, req UserCreateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	if isSuperAdmin(actor.RoleCodes) {
		return s.Create(ctx, req)
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, apperror.Forbidden("当前账号未绑定部门，无法管理用户")
	}
	allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
	if err != nil {
		return nil, err
	}
	if req.DepartmentID == nil {
		deptID := *actor.DepartmentID
		req.DepartmentID = &deptID
	} else if *req.DepartmentID > 0 && !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
		return nil, apperror.Forbidden("无权在该部门下创建用户")
	}
	return s.Create(ctx, req)
}

// Update 更新相关的业务逻辑。
func (s *UserService) Update(ctx context.Context, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
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
	if req.DepartmentID != nil {
		if *req.DepartmentID == 0 {
			user.DepartmentID = nil
		} else {
			if _, err = s.departmentRepo.GetByID(ctx, *req.DepartmentID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, apperror.BadRequest("所属部门不存在")
				}
				return nil, err
			}
			user.DepartmentID = req.DepartmentID
		}
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

func (s *UserService) UpdateByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("无访问权限")
	}
	if !isSuperAdmin(actor.RoleCodes) && req.DepartmentID != nil && *req.DepartmentID > 0 {
		allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
			return nil, apperror.Forbidden("无权将用户调整到目标部门")
		}
	}
	return s.Update(ctx, id, req)
}

// Delete 删除相关的业务逻辑。
func (s *UserService) Delete(ctx context.Context, id uint) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("用户不存在")
		}
		return err
	}

	if err = s.userRepo.Delete(ctx, user); err != nil {
		return err
	}
	_, err = s.enforcer.DeleteUser(UserSubject(id))
	return err
}

func (s *UserService) DeleteByActor(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if actor == nil {
		return apperror.Unauthorized("未登录或登录已失效")
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("用户不存在")
		}
		return err
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.Forbidden("无访问权限")
	}
	return s.Delete(ctx, id)
}

// Detail 查询详情相关的业务逻辑。
func (s *UserService) Detail(ctx context.Context, id uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) DetailByActor(ctx context.Context, actor *auth.CurrentUser, id uint) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("无访问权限")
	}
	resp := NewUserDetailResponse(*target)
	return &resp, nil
}

// List 查询列表相关的业务逻辑。
func (s *UserService) List(ctx context.Context, query UserListQuery) (*pagination.Result[UserDetailResponse], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	users, total, err := s.userRepo.List(ctx, repository.UserListParams{
		Keyword:      query.Keyword,
		DepartmentID: query.DepartmentID,
		Page:         page,
		PageSize:     pageSize,
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

func (s *UserService) ListByActor(ctx context.Context, actor *auth.CurrentUser, query UserListQuery) (*pagination.Result[UserDetailResponse], error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	if isSuperAdmin(actor.RoleCodes) {
		return s.List(ctx, query)
	}
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params := repository.UserListParams{
		Keyword:      query.Keyword,
		DepartmentID: query.DepartmentID,
		Page:         page,
		PageSize:     pageSize,
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		uid := actor.ID
		params.OnlyUserID = &uid
	} else {
		ids, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, err
		}
		params.DepartmentIDs = ids
	}
	users, total, err := s.userRepo.List(ctx, params)
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

// AssignRoles 执行对应的业务逻辑。
func (s *UserService) AssignRoles(ctx context.Context, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, apperror.BadRequest("部分角色不存在")
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

func (s *UserService) AssignRolesByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("无访问权限")
	}
	return s.AssignRoles(ctx, id, req)
}

func (s *UserService) ensureUserUnique(ctx context.Context, currentID uint, username, email string) error {
	if strings.TrimSpace(username) != "" {
		existing, err := s.userRepo.GetByUsername(ctx, strings.TrimSpace(username))
		if err == nil && existing.ID != currentID {
			return apperror.Conflict("用户名已存在")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if strings.TrimSpace(email) != "" {
		existing, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
		if err == nil && existing.ID != currentID {
			return apperror.Conflict("邮箱已存在")
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

func (s *UserService) ListAllByActor(ctx context.Context, actor *auth.CurrentUser) ([]model.User, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	if isSuperAdmin(actor.RoleCodes) {
		return s.ListAll(ctx)
	}
	params := repository.UserListParams{
		Page:     1,
		PageSize: 100000,
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		uid := actor.ID
		params.OnlyUserID = &uid
	} else {
		ids, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, err
		}
		params.DepartmentIDs = ids
	}
	users, _, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	return users, nil
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

		var departmentID *uint
		if len(row) >= 6 {
			deptCode := strings.TrimSpace(row[5])
			if deptCode != "" {
				if dept, findErr := s.departmentRepo.GetByCode(ctx, deptCode); findErr == nil {
					departmentID = &dept.ID
				}
			}
		}

		// skip if user exists
		exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, username, "")
		if err == nil && exists {
			continue
		}

		hashed, _ := password.Hash("123456")
		user := model.User{
			Username:     username,
			Nickname:     nickname,
			Email:        emailPtr,
			Password:     hashed,
			Status:       status,
			DepartmentID: departmentID,
		}
		_ = s.userRepo.Create(ctx, &user)
	}
	return nil
}

// UsersImportTemplateExcel returns the user import template file.
func (s *UserService) UsersImportTemplateExcel() (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]interface{}{"ID(可留空)", "Username(必填)", "Nickname", "Email", "Status(1启用/0停用)", "DepartmentCode(可选)"})
	_ = f.SetSheetRow(sheet, "A2", &[]interface{}{"", "demo.user", "示例用户", "demo@example.com", 1, "RND-PLATFORM"})
	return f, nil
}
