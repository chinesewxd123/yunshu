package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/repository"
)

type K8sNamespaceAllowCreateRequest struct {
	PrincipalKind string `json:"principal_kind" binding:"required"`
	PrincipalRef  string `json:"principal_ref" binding:"required"`
	ClusterID     uint   `json:"cluster_id" binding:"required"`
	Namespace     string `json:"namespace" binding:"required"`
}

type K8sNamespaceAllowService struct {
	repo *repository.K8sNamespaceAllowRepository
}

func NewK8sNamespaceAllowService(repo *repository.K8sNamespaceAllowRepository) *K8sNamespaceAllowService {
	return &K8sNamespaceAllowService{repo: repo}
}

func (s *K8sNamespaceAllowService) List(ctx context.Context, principalKind, principalRef string, clusterID uint) ([]model.K8sNamespaceAllowRule, error) {
	if s.repo == nil {
		return []model.K8sNamespaceAllowRule{}, nil
	}
	return s.repo.List(ctx, principalKind, principalRef, clusterID)
}

func (s *K8sNamespaceAllowService) Create(ctx context.Context, req K8sNamespaceAllowCreateRequest) (*model.K8sNamespaceAllowRule, error) {
	if s.repo == nil {
		return nil, constants.ErrInternal
	}
	k := strings.TrimSpace(strings.ToLower(req.PrincipalKind))
	ref := strings.TrimSpace(req.PrincipalRef)
	ns := strings.TrimSpace(req.Namespace)
	if k == "" || ref == "" || ns == "" {
		return nil, constants.ErrInvalidRequestParam
	}
	if ns == "*" || ns == "_cluster" {
		return nil, constants.ErrBadRequestWithMsg("允许的命名空间不能为 * 或 _cluster")
	}
	it := &model.K8sNamespaceAllowRule{
		PrincipalKind: k,
		PrincipalRef:  ref,
		ClusterID:     req.ClusterID,
		Namespace:     ns,
	}
	if err := s.repo.Create(ctx, it); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, constants.ErrConflictWithMsg("该主体在此集群下对该命名空间的允许规则已存在")
		}
		return nil, svcerr.Pass(ctx, "k8s.namespace-allow", "Create", err)
	}
	return it, nil
}

func (s *K8sNamespaceAllowService) Delete(ctx context.Context, id uint) error {
	if s.repo == nil {
		return constants.ErrInternal
	}
	if id == 0 {
		return constants.ErrBadRequest
	}
	return s.repo.DeleteByID(ctx, id)
}
