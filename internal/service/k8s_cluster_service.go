package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/projectaccess"
	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	corev1 "k8s.io/api/core/v1"
	"gorm.io/gorm"
)

type K8sClusterListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type K8sClusterCreateRequest struct {
	Name              string        `json:"name" binding:"required,max=128"`
	ConnectionMode    string        `json:"connection_mode,omitempty" binding:"omitempty,oneof=kubeconfig direct"`
	Kubeconfig        string        `json:"kubeconfig,omitempty"`
	DirectConfig      *DirectConfig `json:"direct_config,omitempty"`
	OwningProjectID   *uint         `json:"owning_project_id"`
}

type DirectConfig struct {
	Server                string `json:"server" binding:"omitempty,url"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify,omitempty"`
	CAData                string `json:"ca_data,omitempty"`
	Token                 string `json:"token,omitempty"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
	ClientCertData        string `json:"client_cert_data,omitempty"`
	ClientKeyData         string `json:"client_key_data,omitempty"`
	// DictConfigKey 从数据字典读取的配置键
	DictConfigKey string `json:"dict_config_key,omitempty"`
}

type K8sClusterUpdateRequest struct {
	Name              *string       `json:"name,omitempty" binding:"omitempty,max=128"`
	ConnectionMode    *string       `json:"connection_mode,omitempty" binding:"omitempty,oneof=kubeconfig direct"`
	Kubeconfig        *string       `json:"kubeconfig,omitempty"`
	DirectConfig      *DirectConfig `json:"direct_config,omitempty"`
	OwningProjectID   *uint         `json:"owning_project_id"`
}

type K8sClusterSetStatusRequest struct {
	Status int `json:"status" binding:"oneof=0 1"`
}

type K8sClusterItem struct {
	ID                uint          `json:"id"`
	Name              string        `json:"name"`
	OwningProjectID   *uint         `json:"owning_project_id,omitempty"`
	ConnectionMode    string        `json:"connection_mode,omitempty"`
	Kubeconfig     string        `json:"kubeconfig,omitempty"`
	DirectConfig   *DirectConfig `json:"direct_config,omitempty"`
	Status         int           `json:"status"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
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
	repo         *repository.K8sClusterRepository
	dictRepo     *repository.DictEntryRepository
	runtime      *K8sRuntimeService
	nsDenyRepo   *repository.K8sNamespaceDenyRepository
	nsAllowRepo  *repository.K8sNamespaceAllowRepository
	memberRepo   *repository.ProjectMemberRepository
}

// NewK8sClusterService 创建相关逻辑。
func NewK8sClusterService(
	repo *repository.K8sClusterRepository,
	dictRepo *repository.DictEntryRepository,
	runtime *K8sRuntimeService,
	nsDeny *repository.K8sNamespaceDenyRepository,
	nsAllow *repository.K8sNamespaceAllowRepository,
	memberRepo *repository.ProjectMemberRepository,
) *K8sClusterService {
	return &K8sClusterService{
		repo:        repo,
		dictRepo:    dictRepo,
		runtime:     runtime,
		nsDenyRepo:  nsDeny,
		nsAllowRepo: nsAllow,
		memberRepo:   memberRepo,
	}
}

func (s *K8sClusterService) ensureClusterOwningProjectAccess(ctx context.Context, cl *model.K8sCluster) error {
	if cl == nil || cl.OwningProjectID == nil || *cl.OwningProjectID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return nil
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, *cl.OwningProjectID, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrK8sClusterProjectAccessDenied
		}
		return err
	}
	return nil
}

func (s *K8sClusterService) validateAssignOwningProject(ctx context.Context, pid uint) error {
	if pid == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, pid, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrK8sClusterProjectAccessDenied
		}
		return err
	}
	if !projectaccess.RoleAtLeast(m.Role, "admin") {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

// List 查询列表相关的业务逻辑。
func (s *K8sClusterService) List(ctx context.Context, query K8sClusterListQuery) (*K8sClusterListResponse, error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params := repository.K8sClusterListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) && s.memberRepo != nil {
		ids, _ := s.memberRepo.ListProjectIDsByUser(ctx, u.ID)
		params.ProjectMemberFilter = true
		params.ProjectMemberIDs = ids
	}
	clusters, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	items := make([]K8sClusterItem, 0, len(clusters))
	for _, c := range clusters {
		items = append(items, buildClusterItem(c))
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
	// 设置默认连接模式
	connectionMode := req.ConnectionMode
	if connectionMode == "" {
		connectionMode = "kubeconfig"
	}

	c := &model.K8sCluster{
		Name:           req.Name,
		ConnectionMode: connectionMode,
		Status:         1,
	}

	// 处理直连配置
	if connectionMode == "direct" && req.DirectConfig != nil {
		// 如果从字典读取配置
		if req.DirectConfig.DictConfigKey != "" && s.dictRepo != nil {
			dictConfig, err := getDirectConfigFromDict(ctx, s.dictRepo, req.DirectConfig.DictConfigKey)
			if err != nil {
				return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte5d845e17676, err))
			}
			// 合并字典配置和用户配置（用户配置优先）
			mergeDirectConfig(dictConfig, req.DirectConfig)
			req.DirectConfig = dictConfig
		}

		directConfigJSON, err := json.Marshal(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg(constants.ErrMsg2569b002d990)
		}
		c.DirectConfig = string(directConfigJSON)
		// 为直连模式生成兼容的kubeconfig
		kubeconfig, err := buildKubeconfigFromDirectConfig(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmt92e759c1fa53, err))
		}
		c.Kubeconfig = kubeconfig
	} else {
		c.Kubeconfig = req.Kubeconfig
	}

	if req.OwningProjectID != nil && *req.OwningProjectID > 0 {
		if err := s.validateAssignOwningProject(ctx, *req.OwningProjectID); err != nil {
			return nil, err
		}
		v := *req.OwningProjectID
		c.OwningProjectID = &v
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return &K8sClusterItem{
		ID:                c.ID,
		Name:              c.Name,
		OwningProjectID:   c.OwningProjectID,
		ConnectionMode:    c.ConnectionMode,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sClusterService) Detail(ctx context.Context, id uint) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	item := buildClusterItem(*cluster)
	item.Kubeconfig = cluster.Kubeconfig
	return &item, nil
}

// Update 更新相关的业务逻辑。
func (s *K8sClusterService) Update(ctx context.Context, id uint, req K8sClusterUpdateRequest) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	if req.Name != nil {
		cluster.Name = *req.Name
	}

	// 处理连接模式变更
	if req.ConnectionMode != nil {
		cluster.ConnectionMode = *req.ConnectionMode
	}

	// 处理直连配置更新
	if cluster.ConnectionMode == "direct" && req.DirectConfig != nil {
		preserveDirectAuthFromStored(cluster.DirectConfig, req.DirectConfig)
		// 如果从字典读取配置
		if req.DirectConfig.DictConfigKey != "" && s.dictRepo != nil {
			dictConfig, err := getDirectConfigFromDict(ctx, s.dictRepo, req.DirectConfig.DictConfigKey)
			if err != nil {
				return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte5d845e17676, err))
			}
			// 合并字典配置和用户配置（用户配置优先）
			mergeDirectConfig(dictConfig, req.DirectConfig)
			req.DirectConfig = dictConfig
		}

		directConfigJSON, err := json.Marshal(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg(constants.ErrMsg2569b002d990)
		}
		cluster.DirectConfig = string(directConfigJSON)

		// 为直连模式生成兼容的kubeconfig
		kubeconfig, err := buildKubeconfigFromDirectConfig(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmt92e759c1fa53, err))
		}
		cluster.Kubeconfig = kubeconfig
		s.runtime.DeleteRegisterCache(cluster.ID)
	} else if req.Kubeconfig != nil {
		cluster.Kubeconfig = *req.Kubeconfig
		s.runtime.DeleteRegisterCache(cluster.ID)
	}

	if req.OwningProjectID != nil {
		if *req.OwningProjectID == 0 {
			cluster.OwningProjectID = nil
		} else {
			if err := s.validateAssignOwningProject(ctx, *req.OwningProjectID); err != nil {
				return nil, err
			}
			v := *req.OwningProjectID
			cluster.OwningProjectID = &v
		}
	}

	if err := s.repo.Update(ctx, cluster); err != nil {
		return nil, err
	}
	out := buildClusterItem(*cluster)
	return &out, nil
}

// Delete 删除相关的业务逻辑。
func (s *K8sClusterService) Delete(ctx context.Context, id uint) error {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return err
	}
	s.runtime.DeleteRegisterCache(id)
	return s.repo.Delete(ctx, id)
}

// SetStatus 设置相关的业务逻辑。
func (s *K8sClusterService) SetStatus(ctx context.Context, id uint, status int) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	if status != 0 && status != 1 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg394db01d16f3)
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
	out := buildClusterItem(*cluster)
	return &out, nil
}

func buildClusterItem(c model.K8sCluster) K8sClusterItem {
	item := K8sClusterItem{
		ID:                c.ID,
		Name:              c.Name,
		OwningProjectID:   c.OwningProjectID,
		ConnectionMode:    c.ConnectionMode,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
	if c.ConnectionMode == "direct" && strings.TrimSpace(c.DirectConfig) != "" {
		var dc DirectConfig
		if err := json.Unmarshal([]byte(c.DirectConfig), &dc); err == nil {
			item.DirectConfig = &dc
		}
	}
	return item
}

// Status 执行对应的业务逻辑。
func (s *K8sClusterService) Status(ctx context.Context, id uint) (*K8sClusterStatusResponse, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
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

// ListNamespaces 查询列表相关的业务逻辑；若传入 pack，则按命名空间黑/白名单过滤（与控制台下拉、K8sScopeAuthorize 对齐）。
func (s *K8sClusterService) ListNamespaces(ctx context.Context, id uint, pack *k8sauth.PrincipalPack) ([]NamespaceItem, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
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
	if pack == nil || len(pack.PrincipalRows()) == 0 {
		return out, nil
	}
	names := make([]string, len(out))
	for i := range out {
		names[i] = out[i].Name
	}
	names, err = FilterNamespaceNamesByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, *pack, id, names)
	if err != nil {
		return nil, err
	}
	keep := make(map[string]struct{}, len(names))
	for _, n := range names {
		keep[n] = struct{}{}
	}
	filtered := out[:0]
	for _, it := range out {
		if _, ok := keep[it.Name]; ok {
			filtered = append(filtered, it)
		}
	}
	return filtered, nil
}

// ListComponentStatuses 查询列表相关的业务逻辑。
func (s *K8sClusterService) ListComponentStatuses(ctx context.Context, id uint) ([]ComponentStatusItem, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, id)
	if err != nil {
		return nil, err
	}
	probedAt := time.Now().Format(time.RFC3339)
	var list []corev1.ComponentStatus
	if err := k.Resource(&corev1.ComponentStatus{}).List(&list).Error; err != nil {
		return nil, constants.ErrInternalWithMsg(fmt.Sprintf(constants.ErrFmt559cb56d5b9d, err))
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
