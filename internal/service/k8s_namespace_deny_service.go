package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/repository"
)

type K8sNamespaceDenyCreateRequest struct {
	RoleCode  string `json:"role_code" binding:"required"`
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
}

type K8sNamespaceDenyService struct {
	repo *repository.K8sNamespaceDenyRepository
}

func NewK8sNamespaceDenyService(repo *repository.K8sNamespaceDenyRepository) *K8sNamespaceDenyService {
	return &K8sNamespaceDenyService{repo: repo}
}

func (s *K8sNamespaceDenyService) List(ctx context.Context, roleCode string, clusterID uint) ([]model.K8sNamespaceDenyRule, error) {
	if s.repo == nil {
		return []model.K8sNamespaceDenyRule{}, nil
	}
	return s.repo.List(ctx, roleCode, clusterID)
}

func (s *K8sNamespaceDenyService) Create(ctx context.Context, req K8sNamespaceDenyCreateRequest) (*model.K8sNamespaceDenyRule, error) {
	if s.repo == nil {
		return nil, constants.ErrInternal
	}
	rc := strings.TrimSpace(req.RoleCode)
	ns := strings.TrimSpace(req.Namespace)
	if rc == "" || ns == "" {
		return nil, constants.ErrInvalidRequestParam
	}
	if ns == "*" || ns == "_cluster" {
		return nil, constants.ErrBadRequestWithMsg("禁止的命名空间不能为 * 或 _cluster")
	}
	it := &model.K8sNamespaceDenyRule{
		RoleCode:  rc,
		ClusterID: req.ClusterID,
		Namespace: ns,
	}
	if err := s.repo.Create(ctx, it); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, constants.ErrConflictWithMsg("该角色在此集群下对该命名空间的禁止规则已存在")
		}
		return nil, err
	}
	return it, nil
}

func (s *K8sNamespaceDenyService) Delete(ctx context.Context, id uint) error {
	if s.repo == nil {
		return constants.ErrInternal
	}
	if id == 0 {
		return constants.ErrBadRequest
	}
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return err
	}
	return nil
}
