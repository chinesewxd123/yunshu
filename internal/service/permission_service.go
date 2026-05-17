package service

import (
	"context"
	"errors"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type PermissionService struct {
	permissionRepo *repository.PermissionRepository
	enforcer       *casbin.SyncedEnforcer
}

// NewPermissionService 创建相关逻辑。
func NewPermissionService(permissionRepo *repository.PermissionRepository, enforcer *casbin.SyncedEnforcer) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
		enforcer:       enforcer,
	}
}

// Create 创建相关的业务逻辑。
func (s *PermissionService) Create(ctx context.Context, req PermissionCreateRequest) (*PermissionItem, error) {
	permission := model.Permission{
		Name:            req.Name,
		Resource:        req.Resource,
		Action:          req.Action,
		Description:     req.Description,
		K8sScopeEnabled: req.K8sScopeEnabled,
	}
	if err := s.permissionRepo.Create(ctx, &permission); err != nil {
		return nil, svcerr.Pass("permission", "Create", err)
	}
	response := NewPermissionItem(permission)
	return &response, nil
}

// Update 更新相关的业务逻辑。
func (s *PermissionService) Update(ctx context.Context, id uint, req PermissionUpdateRequest) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrPermissionNotFound
		}
		return nil, svcerr.Pass("permission", "Update", err)
	}

	oldResource := permission.Resource
	oldAction := permission.Action
	if req.Name != nil {
		permission.Name = *req.Name
	}
	if req.Resource != nil {
		permission.Resource = *req.Resource
	}
	if req.Action != nil {
		permission.Action = *req.Action
	}
	if req.Description != nil {
		permission.Description = *req.Description
	}
	if req.K8sScopeEnabled != nil {
		permission.K8sScopeEnabled = *req.K8sScopeEnabled
	}

	if err = s.permissionRepo.Save(ctx, permission); err != nil {
		return nil, svcerr.Pass("permission", "Update", err)
	}
	if err = ReplacePermissionResource(s.enforcer, oldResource, oldAction, permission.Resource, permission.Action); err != nil {
		return nil, svcerr.Pass("permission", "Update", err)
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

// Delete 删除相关的业务逻辑。
func (s *PermissionService) Delete(ctx context.Context, id uint) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrPermissionNotFound
		}
		return svcerr.Pass("permission", "Delete", err)
	}

	if err = s.permissionRepo.Delete(ctx, permission); err != nil {
		return svcerr.Pass("permission", "Delete", err)
	}
	return RemovePermissionPolicies(s.enforcer, permission.Resource, permission.Action)
}

// Detail 查询详情相关的业务逻辑。
func (s *PermissionService) Detail(ctx context.Context, id uint) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrPermissionNotFound
		}
		return nil, svcerr.Pass("permission", "Detail", err)
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

// List 查询列表相关的业务逻辑。
func (s *PermissionService) List(ctx context.Context, query PermissionListQuery) (*pagination.Result[PermissionItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	permissions, total, err := s.permissionRepo.List(ctx, repository.PermissionListParams{
		Keyword:    query.Keyword,
		Page:       page,
		PageSize:   pageSize,
		K8sScope:   query.K8sScope,
		K8sRelated: query.K8sRelated,
	})
	if err != nil {
		return nil, svcerr.Pass("permission", "List", err)
	}

	list := make([]PermissionItem, 0, len(permissions))
	for _, permission := range permissions {
		list = append(list, NewPermissionItem(permission))
	}

	return &pagination.Result[PermissionItem]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
