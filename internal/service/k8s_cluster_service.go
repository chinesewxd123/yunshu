package service

import (
	"context"
	"fmt"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/repository"

	corev1 "k8s.io/api/core/v1"
)

type K8sClusterListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type K8sClusterCreateRequest struct {
	Name       string `json:"name" binding:"required,max=128"`
	Kubeconfig string `json:"kubeconfig" binding:"required"`
}

type K8sClusterUpdateRequest struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,max=128"`
	Kubeconfig *string `json:"kubeconfig,omitempty" binding:"omitempty"`
}

type K8sClusterSetStatusRequest struct {
	Status int `json:"status" binding:"oneof=0 1"`
}

type K8sClusterItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type K8sClusterListResponse struct {
	List     []K8sClusterItem `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type K8sClusterStatusResponse struct {
	ServerVersion string `json:"server_version"`
}

type NamespaceItem struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}

type K8sClusterService struct {
	repo    *repository.K8sClusterRepository
	runtime *K8sRuntimeService
}

func NewK8sClusterService(repo *repository.K8sClusterRepository, runtime *K8sRuntimeService) *K8sClusterService {
	return &K8sClusterService{
		repo:    repo,
		runtime: runtime,
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (s *K8sClusterService) List(ctx context.Context, query K8sClusterListQuery) (*K8sClusterListResponse, error) {
	clusters, total, err := s.repo.List(ctx, repository.K8sClusterListParams{
		Keyword:  query.Keyword,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	items := make([]K8sClusterItem, 0, len(clusters))
	for _, c := range clusters {
		items = append(items, K8sClusterItem{
			ID:        c.ID,
			Name:      c.Name,
			Status:    c.Status,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return &K8sClusterListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *K8sClusterService) Create(ctx context.Context, req K8sClusterCreateRequest) (*K8sClusterItem, error) {
	c := &model.K8sCluster{Name: req.Name, Kubeconfig: req.Kubeconfig, Status: 1}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return &K8sClusterItem{ID: c.ID, Name: c.Name, Status: c.Status, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}, nil
}

func (s *K8sClusterService) Update(ctx context.Context, id uint, req K8sClusterUpdateRequest) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		cluster.Name = *req.Name
	}
	if req.Kubeconfig != nil {
		cluster.Kubeconfig = *req.Kubeconfig
		s.runtime.DeleteRegisterCache(cluster.ID)
	}
	if err := s.repo.Update(ctx, cluster); err != nil {
		return nil, err
	}
	return &K8sClusterItem{ID: cluster.ID, Name: cluster.Name, Status: cluster.Status, CreatedAt: cluster.CreatedAt, UpdatedAt: cluster.UpdatedAt}, nil
}

func (s *K8sClusterService) Delete(ctx context.Context, id uint) error {
	s.runtime.DeleteRegisterCache(id)
	return s.repo.Delete(ctx, id)
}

func (s *K8sClusterService) SetStatus(ctx context.Context, id uint, status int) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if status != 0 && status != 1 {
		return nil, apperror.BadRequest("status 只能为 0 或 1")
	}
	if cluster.Status != status {
		cluster.Status = status
		// 停用后立即让 kom 重新注册失效，避免继续复用旧连接
		if status == 0 {
			s.runtime.DeleteRegisterCache(id)
		}
		if err := s.repo.Update(ctx, cluster); err != nil {
			return nil, err
		}
	}
	return &K8sClusterItem{ID: cluster.ID, Name: cluster.Name, Status: cluster.Status, CreatedAt: cluster.CreatedAt, UpdatedAt: cluster.UpdatedAt}, nil
}

func (s *K8sClusterService) Status(ctx context.Context, id uint) (*K8sClusterStatusResponse, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, id)
	if err != nil {
		return nil, err
	}
	info := k.Status().ServerVersion()
	var lastErr error
	if info == nil || info.GitVersion == "" {
		if client := k.Client(); client != nil {
			fresh, e := client.Discovery().ServerVersion()
			if e != nil {
				lastErr = e
			}
			if e == nil && fresh != nil {
				info = fresh
			}
		}
	}
	ver := ""
	if info != nil {
		ver = info.GitVersion
	}
	if ver == "" {
		if lastErr != nil {
			return nil, apperror.Internal(fmt.Sprintf("获取 K8s 版本失败: %v", lastErr))
		}
		return nil, apperror.Internal("获取 K8s 版本失败: 版本信息为空")
	}
	return &K8sClusterStatusResponse{ServerVersion: ver}, nil
}

func (s *K8sClusterService) ListNamespaces(ctx context.Context, id uint) ([]NamespaceItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, id)
	if err != nil {
		return nil, err
	}
	var nsList []corev1.Namespace
	if err := k.Resource(&corev1.Namespace{}).List(&nsList).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceItem, 0, len(nsList))
	for _, ns := range nsList {
		out = append(out, NamespaceItem{Name: ns.Name, Phase: string(ns.Status.Phase)})
	}
	return out, nil
}
