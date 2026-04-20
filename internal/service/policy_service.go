package service

import (
	"context"
	"errors"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type PolicyService struct {
	roleRepo       *repository.RoleRepository
	permissionRepo *repository.PermissionRepository
	enforcer       *casbin.SyncedEnforcer
}

// NewPolicyService 创建相关逻辑。
func NewPolicyService(roleRepo *repository.RoleRepository, permissionRepo *repository.PermissionRepository, enforcer *casbin.SyncedEnforcer) *PolicyService {
	return &PolicyService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		enforcer:       enforcer,
	}
}

// List 查询列表相关的业务逻辑。
func (s *PolicyService) List(ctx context.Context) ([]PolicyItemResponse, error) {
	roles, err := s.roleRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	permissions, err := s.permissionRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	roleMap := make(map[string]RoleItem, len(roles))
	for _, role := range roles {
		roleMap[role.Code] = NewRoleItem(role)
	}

	permissionMap := make(map[string]PermissionItem, len(permissions))
	for _, permission := range permissions {
		key := permission.Resource + "::" + permission.Action
		permissionMap[key] = NewPermissionItem(permission)
	}

	policies := s.enforcer.GetPolicy()

	response := make([]PolicyItemResponse, 0, len(policies))
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}

		role := roleMap[policy[0]]
		permission := permissionMap[policy[1]+"::"+policy[2]]
		response = append(response, PolicyItemResponse{
			RoleID:         role.ID,
			RoleName:       role.Name,
			RoleCode:       role.Code,
			PermissionID:   permission.ID,
			PermissionName: permission.Name,
			Resource:       policy[1],
			Action:         policy[2],
		})
	}
	return response, nil
}

// Grant 执行对应的业务逻辑。
func (s *PolicyService) Grant(ctx context.Context, req PolicyGrantRequest) error {
	role, err := s.roleRepo.GetByID(ctx, req.RoleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("角色不存在")
		}
		return err
	}

	permission, err := s.permissionRepo.GetByID(ctx, req.PermissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("权限不存在")
		}
		return err
	}

	_, err = s.enforcer.AddPolicy(role.Code, permission.Resource, permission.Action)
	return err
}

// Revoke 执行对应的业务逻辑。
func (s *PolicyService) Revoke(ctx context.Context, req PolicyGrantRequest) error {
	role, err := s.roleRepo.GetByID(ctx, req.RoleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("角色不存在")
		}
		return err
	}

	permission, err := s.permissionRepo.GetByID(ctx, req.PermissionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("权限不存在")
		}
		return err
	}

	_, err = s.enforcer.RemovePolicy(role.Code, permission.Resource, permission.Action)
	return err
}
