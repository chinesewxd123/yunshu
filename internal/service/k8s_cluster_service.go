package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"
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
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Kubeconfig string    `json:"kubeconfig,omitempty"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type K8sClusterListResponse struct {
	List     []K8sClusterItem `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type K8sClusterStatusResponse struct {
	ServerVersion       string    `json:"server_version"`
	ConnectionState     string    `json:"connection_state"`
	LastError           string    `json:"last_error,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

type NamespaceItem struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}

type ComponentStatusItem struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Healthy     bool   `json:"healthy"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
	LastProbeAt string `json:"last_probe_at,omitempty"`
}

type K8sClusterService struct {
	repo    *repository.K8sClusterRepository
	runtime *K8sRuntimeService
}

// NewK8sClusterService 创建相关逻辑。
func NewK8sClusterService(repo *repository.K8sClusterRepository, runtime *K8sRuntimeService) *K8sClusterService {
	return &K8sClusterService{
		repo:    repo,
		runtime: runtime,
	}
}

// List 查询列表相关的业务逻辑。
func (s *K8sClusterService) List(ctx context.Context, query K8sClusterListQuery) (*K8sClusterListResponse, error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	clusters, total, err := s.repo.List(ctx, repository.K8sClusterListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
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

// Create 创建相关的业务逻辑。
func (s *K8sClusterService) Create(ctx context.Context, req K8sClusterCreateRequest) (*K8sClusterItem, error) {
	c := &model.K8sCluster{Name: req.Name, Kubeconfig: req.Kubeconfig, Status: 1}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return &K8sClusterItem{ID: c.ID, Name: c.Name, Status: c.Status, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sClusterService) Detail(ctx context.Context, id uint) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &K8sClusterItem{
		ID:         cluster.ID,
		Name:       cluster.Name,
		Kubeconfig: cluster.Kubeconfig,
		Status:     cluster.Status,
		CreatedAt:  cluster.CreatedAt,
		UpdatedAt:  cluster.UpdatedAt,
	}, nil
}

// Update 更新相关的业务逻辑。
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

// Delete 删除相关的业务逻辑。
func (s *K8sClusterService) Delete(ctx context.Context, id uint) error {
	s.runtime.DeleteRegisterCache(id)
	return s.repo.Delete(ctx, id)
}

// SetStatus 设置相关的业务逻辑。
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

// Status 执行对应的业务逻辑。
func (s *K8sClusterService) Status(ctx context.Context, id uint) (*K8sClusterStatusResponse, error) {
	ver, state, err := s.runtime.CheckClusterHeartbeat(ctx, id)
	if err != nil {
		return nil, err
	}
	return &K8sClusterStatusResponse{
		ServerVersion:       ver,
		ConnectionState:     state.State,
		LastError:           state.LastError,
		LastAttemptAt:       state.LastAttemptAt,
		LastSuccessAt:       state.LastSuccessAt,
		ConsecutiveFailures: state.ConsecutiveFailures,
	}, nil
}

// ListNamespaces 查询列表相关的业务逻辑。
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

// ListComponentStatuses 查询列表相关的业务逻辑。
func (s *K8sClusterService) ListComponentStatuses(ctx context.Context, id uint) ([]ComponentStatusItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, id)
	if err != nil {
		return nil, err
	}
	probedAt := time.Now().Format(time.RFC3339)
	var list []corev1.ComponentStatus
	if err := k.Resource(&corev1.ComponentStatus{}).List(&list).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取组件状态失败: %v", err))
	}
	out := make([]ComponentStatusItem, 0, len(list))
	for _, item := range list {
		state := "Unknown"
		healthy := false
		message := ""
		reason := ""
		for _, cond := range item.Conditions {
			if cond.Type != corev1.ComponentHealthy {
				continue
			}
			switch cond.Status {
			case corev1.ConditionTrue:
				state = "Healthy"
				healthy = true
			case corev1.ConditionFalse:
				state = "Unhealthy"
			default:
				state = "Unknown"
			}
			message = strings.TrimSpace(cond.Message)
			reason = strings.TrimSpace(cond.Error)
			break
		}
		out = append(out, ComponentStatusItem{
			Name:        item.Name,
			Status:      state,
			Healthy:     healthy,
			Message:     message,
			Error:       reason,
			LastProbeAt: probedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
