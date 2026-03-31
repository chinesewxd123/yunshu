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

type PermissionService struct {
	permissionRepo *repository.PermissionRepository
	enforcer       *casbin.SyncedEnforcer
}

func NewPermissionService(permissionRepo *repository.PermissionRepository, enforcer *casbin.SyncedEnforcer) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
		enforcer:       enforcer,
	}
}

func (s *PermissionService) Create(ctx context.Context, req PermissionCreateRequest) (*PermissionItem, error) {
	permission := model.Permission{
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}
	if err := s.permissionRepo.Create(ctx, &permission); err != nil {
		return nil, err
	}
	response := NewPermissionItem(permission)
	return &response, nil
}

func (s *PermissionService) Update(ctx context.Context, id uint, req PermissionUpdateRequest) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("permission not found")
		}
		return nil, err
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

	if err = s.permissionRepo.Save(ctx, permission); err != nil {
		return nil, err
	}
	if err = ReplacePermissionResource(s.enforcer, oldResource, oldAction, permission.Resource, permission.Action); err != nil {
		return nil, err
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

func (s *PermissionService) Delete(ctx context.Context, id uint) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("permission not found")
		}
		return err
	}

	if err = s.permissionRepo.Delete(ctx, permission); err != nil {
		return err
	}
	return RemovePermissionPolicies(s.enforcer, permission.Resource, permission.Action)
}

func (s *PermissionService) Detail(ctx context.Context, id uint) (*PermissionItem, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("permission not found")
		}
		return nil, err
	}
	response := NewPermissionItem(*permission)
	return &response, nil
}

func (s *PermissionService) List(ctx context.Context, query PermissionListQuery) (*pagination.Result[PermissionItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	permissions, total, err := s.permissionRepo.List(ctx, repository.PermissionListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
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
