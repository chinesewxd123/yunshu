package service

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo         *repository.UserRepository
	roleRepo         *repository.RoleRepository
	departmentRepo   *repository.DepartmentRepository
	projectMemberRepo *repository.ProjectMemberRepository
	assigneeSvc      *AlertRuleAssigneeService
	enforcer         *casbin.SyncedEnforcer
}

// NewUserService 创建相关逻辑。
func NewUserService(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	departmentRepo *repository.DepartmentRepository,
	enforcer *casbin.SyncedEnforcer,
	projectMemberRepo *repository.ProjectMemberRepository,
	assigneeSvc *AlertRuleAssigneeService,
) *UserService {
	return &UserService{
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		departmentRepo:    departmentRepo,
		projectMemberRepo: projectMemberRepo,
		assigneeSvc:       assigneeSvc,
		enforcer:          enforcer,
	}
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
	if auth.IsSuperAdminRole(actor.RoleCodes) || actor.ID == target.ID {
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
		return false, svcerr.Pass("user", "canAccessUser", err)
	}
	return slices.Contains(ids, *target.DepartmentID), nil
}

// Create 创建相关的业务逻辑。
func (s *UserService) Create(ctx context.Context, req UserCreateRequest) (*UserDetailResponse, error) {
	if err := s.ensureUserUnique(ctx, 0, req.Username, req.Email); err != nil {
		return nil, svcerr.Pass("user", "Create", err)
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, svcerr.Pass("user", "Create", err)
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgbc90b8ad5f29)
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, svcerr.Pass("user", "Create", err)
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
				return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg94d63d947b0e)
			}
			return nil, svcerr.Pass("user", "Create", err)
		}
		departmentID = req.DepartmentID
	}
	user := model.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        &email,
		Phone:        strings.TrimSpace(req.Phone),
		Password:     hashedPassword,
		Nickname:     strings.TrimSpace(req.Nickname),
		Status:       status,
		DepartmentID: departmentID,
		Roles:        roles,
	}
	if err = s.userRepo.Create(ctx, &user); err != nil {
		return nil, svcerr.Pass("user", "Create", err)
	}
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, svcerr.Pass("user", "Create", err)
	}

	response := NewUserDetailResponse(user)
	return &response, nil
}

func (s *UserService) CreateByActor(ctx context.Context, actor *auth.CurrentUser, req UserCreateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.Create(ctx, req)
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgc8caf91c1d57)
	}
	allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
	if err != nil {
		return nil, svcerr.Pass("user", "CreateByActor", err)
	}
	if req.DepartmentID == nil {
		deptID := *actor.DepartmentID
		req.DepartmentID = &deptID
	} else if *req.DepartmentID > 0 && !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
		return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgd672e80435d4)
	}
	return s.Create(ctx, req)
}

// Update 更新相关的业务逻辑。
func (s *UserService) Update(ctx context.Context, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "Update", err)
	}

	if req.Nickname != nil {
		user.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		email := normalizeEmail(*req.Email)
		if err = s.ensureUserUnique(ctx, user.ID, user.Username, email); err != nil {
			return nil, svcerr.Pass("user", "Update", err)
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
					return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg94d63d947b0e)
				}
				return nil, svcerr.Pass("user", "Update", err)
			}
			user.DepartmentID = req.DepartmentID
		}
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Password != nil && *req.Password != "" {
		user.Password, err = password.Hash(*req.Password)
		if err != nil {
			return nil, svcerr.Pass("user", "Update", err)
		}
	}

	if err = s.userRepo.Save(ctx, user); err != nil {
		return nil, svcerr.Pass("user", "Update", err)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) UpdateByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserUpdateRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "UpdateByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, svcerr.Pass("user", "UpdateByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		if actor.ID == target.ID {
			return nil, constants.ErrForbiddenWithMsg("不能通过用户管理修改自己的登录密码")
		}
		if !auth.CanManageOtherUsersLoginPassword(actor.RoleCodes) {
			return nil, constants.ErrForbiddenWithMsg("仅管理员角色（内置超级管理员）可修改其他账号的登录密码")
		}
	}
	if !auth.IsSuperAdminRole(actor.RoleCodes) && req.DepartmentID != nil && *req.DepartmentID > 0 {
		allowedDepartmentIDs, err := s.accessibleDepartmentIDs(ctx, actor)
		if err != nil {
			return nil, svcerr.Pass("user", "UpdateByActor", err)
		}
		if !slices.Contains(allowedDepartmentIDs, *req.DepartmentID) {
			return nil, constants.ErrForbiddenWithMsg(constants.ErrMsgc1305dfff708)
		}
	}
	return s.Update(ctx, id, req)
}

// Delete 删除相关的业务逻辑。
func (s *UserService) Delete(ctx context.Context, id uint) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return svcerr.Pass("user", "Delete", err)
	}

	if err = s.userRepo.Delete(ctx, user); err != nil {
		return svcerr.Pass("user", "Delete", err)
	}
	if s.departmentRepo != nil {
		_ = s.departmentRepo.ClearLeaderByUserID(ctx, id)
	}
	if s.projectMemberRepo != nil {
		_ = s.projectMemberRepo.DeleteByUserID(ctx, id)
	}
	if s.assigneeSvc != nil {
		_ = s.assigneeSvc.PruneUserFromAllAssignees(ctx, id)
	}
	_, err = s.enforcer.DeleteUser(UserSubject(id))
	if err != nil {
		return svcerr.Pass("user", "Delete", err)
	}
	return nil
}

func (s *UserService) DeleteByActor(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return svcerr.Pass("user", "DeleteByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return svcerr.Pass("user", "DeleteByActor", err)
	}
	if !ok {
		return constants.ErrForbidden
	}
	return s.Delete(ctx, id)
}

// Detail 查询详情相关的业务逻辑。
func (s *UserService) Detail(ctx context.Context, id uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "Detail", err)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) DetailByActor(ctx context.Context, actor *auth.CurrentUser, id uint) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "DetailByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, svcerr.Pass("user", "DetailByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
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
		return nil, svcerr.Pass("user", "List", err)
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
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
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
			return nil, svcerr.Pass("user", "ListByActor", err)
		}
		params.DepartmentIDs = ids
	}
	users, total, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, svcerr.Pass("user", "ListByActor", err)
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
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "AssignRoles", err)
	}

	roles, err := s.roleRepo.GetByIDs(ctx, req.RoleIDs)
	if err != nil {
		return nil, svcerr.Pass("user", "AssignRoles", err)
	}
	if len(req.RoleIDs) > 0 && len(roles) != len(req.RoleIDs) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgbc90b8ad5f29)
	}

	if err = s.userRepo.ReplaceRoles(ctx, user, roles); err != nil {
		return nil, svcerr.Pass("user", "AssignRoles", err)
	}

	user.Roles = roles
	if err = SyncUserRoles(s.enforcer, user.ID, roles); err != nil {
		return nil, svcerr.Pass("user", "AssignRoles", err)
	}
	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *UserService) AssignRolesByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req UserAssignRolesRequest) (*UserDetailResponse, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("user", "AssignRolesByActor", err)
	}
	ok, err := s.canAccessUser(ctx, actor, target)
	if err != nil {
		return nil, svcerr.Pass("user", "AssignRolesByActor", err)
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	return s.AssignRoles(ctx, id, req)
}

func (s *UserService) ensureUserUnique(ctx context.Context, currentID uint, username, email string) error {
	if strings.TrimSpace(username) != "" {
		existing, err := s.userRepo.GetByUsername(ctx, strings.TrimSpace(username))
		if err == nil && existing.ID != currentID {
			return constants.ErrUsernameTaken
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerr.Pass("user", "ensureUserUnique", err)
		}
	}

	if strings.TrimSpace(email) != "" {
		existing, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
		if err == nil && existing.ID != currentID {
			return constants.ErrEmailAlreadyRegistered
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerr.Pass("user", "ensureUserUnique", err)
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
		return nil, constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
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
			return nil, svcerr.Pass("user", "ListAllByActor", err)
		}
		params.DepartmentIDs = ids
	}
	users, _, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, svcerr.Pass("user", "ListAllByActor", err)
	}
	return users, nil
}

// ImportUsers reads an Excel file from reader and creates users.
func (s *UserService) ImportUsers(ctx context.Context, r io.Reader) error {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return svcerr.Pass("user", "ImportUsers", err)
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return svcerr.Pass("user", "ImportUsers", err)
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
