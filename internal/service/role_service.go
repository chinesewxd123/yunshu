package service

import (
	"context"
	"errors"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type RoleService struct {
	roleRepo *repository.RoleRepository
	enforcer *casbin.SyncedEnforcer
}

// NewRoleService 创建相关逻辑。
func NewRoleService(roleRepo *repository.RoleRepository, enforcer *casbin.SyncedEnforcer) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		enforcer: enforcer,
	}
}

// Create 创建相关的业务逻辑。
func (s *RoleService) Create(ctx context.Context, req RoleCreateRequest) (*RoleItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	role := model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      status,
	}
	if err := s.roleRepo.Create(ctx, &role); err != nil {
		return nil, err
	}
	response := NewRoleItem(role)
	return &response, nil
}

// Update 更新相关的业务逻辑。
func (s *RoleService) Update(ctx context.Context, id uint, req RoleUpdateRequest) (*RoleItem, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("角色不存在")
		}
		return nil, err
	}

	oldCode := role.Code
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		role.Code = *req.Code
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}

	if err = s.roleRepo.Save(ctx, role); err != nil {
		return nil, err
	}
	if err = ReplaceRoleCode(s.enforcer, oldCode, role.Code); err != nil {
		return nil, err
	}
	response := NewRoleItem(*role)
	return &response, nil
}

// Delete 删除相关的业务逻辑。
func (s *RoleService) Delete(ctx context.Context, id uint) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("角色不存在")
		}
		return err
	}

	if err = s.roleRepo.Delete(ctx, role); err != nil {
		return err
	}
	return RemoveRolePolicies(s.enforcer, role.Code)
}

// Detail 查询详情相关的业务逻辑。
func (s *RoleService) Detail(ctx context.Context, id uint) (*RoleItem, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("角色不存在")
		}
		return nil, err
	}
	response := NewRoleItem(*role)
	return &response, nil
}

// List 查询列表相关的业务逻辑。
func (s *RoleService) List(ctx context.Context, query RoleListQuery) (*pagination.Result[RoleItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	roles, total, err := s.roleRepo.List(ctx, repository.RoleListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	list := make([]RoleItem, 0, len(roles))
	for _, role := range roles {
		list = append(list, NewRoleItem(role))
	}

	return &pagination.Result[RoleItem]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
