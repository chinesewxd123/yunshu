package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type K8sActionItem struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// K8sClusterAccessItem 集群档位（DB），不经 Casbin。
type K8sClusterAccessItem struct {
	ID            uint   `json:"id"`
	PrincipalKind string `json:"principal_kind"`
	PrincipalRef  string `json:"principal_ref"`
	RoleCode      string `json:"role_code"` // 兼容：principal_kind=role 时与 principal_ref 相同
	ClusterID     uint   `json:"cluster_id"`
	Preset        string `json:"preset"`
}

func k8sClusterAccessItemFromGrant(g model.K8sClusterAccessGrant) K8sClusterAccessItem {
	it := K8sClusterAccessItem{
		ID:            g.ID,
		PrincipalKind: g.PrincipalKind,
		PrincipalRef:  g.PrincipalRef,
		ClusterID:     g.ClusterID,
		Preset:        g.Preset,
	}
	if strings.EqualFold(strings.TrimSpace(g.PrincipalKind), model.K8sPrincipalRole) {
		it.RoleCode = g.PrincipalRef
	}
	return it
}

// K8sScopedPolicyGrantPresetRequest 下发档位；兼容仅传 role_id（视为 role 主体）。
type K8sScopedPolicyGrantPresetRequest struct {
	PrincipalKind string `json:"principal_kind"` // role|user|group，可空（由 role_id/user_id/group_id 推断）
	RoleID        uint   `json:"role_id"`
	UserID        uint   `json:"user_id"`
	GroupID       uint   `json:"group_id"`
	ClusterIDs    []uint `json:"cluster_ids"`
	Preset        string `json:"preset" binding:"required"` // readonly | readonly_exec | admin
	// 仅对具体集群 ID 写入（cluster_id=0 全部集群时不写）
	DenyNamespaces  []string `json:"deny_namespaces"`
	AllowNamespaces []string `json:"allow_namespaces"`
}

type K8sScopedPolicyGrantPresetResponse struct {
	Added             int `json:"added"`
	Skipped           int `json:"skipped"`
	DenyRulesAdded    int `json:"deny_rules_added"`
	DenyRulesSkipped  int `json:"deny_rules_skipped"`
	AllowRulesAdded   int `json:"allow_rules_added"`
	AllowRulesSkipped int `json:"allow_rules_skipped"`
}

type K8sScopedPolicyService struct {
	roleRepo       *repository.RoleRepository
	permRepo       *repository.PermissionRepository
	accessRepo     *repository.K8sClusterAccessRepository
	nsDenyRepo     *repository.K8sNamespaceDenyRepository
	nsAllowRepo    *repository.K8sNamespaceAllowRepository
	userGroupRepo  *repository.UserGroupRepository
}

// NewK8sScopedPolicyService 创建 K8s 集群档位服务（不写 Casbin k8s: 策略）。
func NewK8sScopedPolicyService(
	roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	accessRepo *repository.K8sClusterAccessRepository,
	nsDenyRepo *repository.K8sNamespaceDenyRepository,
	nsAllowRepo *repository.K8sNamespaceAllowRepository,
	userGroupRepo *repository.UserGroupRepository,
) *K8sScopedPolicyService {
	return &K8sScopedPolicyService{
		roleRepo:      roleRepo,
		permRepo:      permRepo,
		accessRepo:    accessRepo,
		nsDenyRepo:     nsDenyRepo,
		nsAllowRepo:    nsAllowRepo,
		userGroupRepo: userGroupRepo,
	}
}

// ActionCatalog 动作码目录（供文档/展示；档位授权不再逐条绑定动作码）。
func (s *K8sScopedPolicyService) ActionCatalog() []K8sActionItem {
	if s.permRepo == nil {
		return []K8sActionItem{}
	}
	perms, err := s.permRepo.ListAll(context.Background())
	if err != nil {
		return []K8sActionItem{}
	}
	actions, _ := BuildK8sScopeActionCatalog(perms)
	return actions
}

// PathCatalog 纳入 K8s 范围校验的 API 路径目录。
func (s *K8sScopedPolicyService) PathCatalog() []string {
	if s.permRepo == nil {
		return []string{}
	}
	perms, err := s.permRepo.ListAll(context.Background())
	if err != nil {
		return []string{}
	}
	_, paths := BuildK8sScopeActionCatalog(perms)
	return paths
}

// GrantPreset 按 k8m 风格写入 k8s_cluster_access_grants（主体可为角色 / 用户 / 组）。
func (s *K8sScopedPolicyService) GrantPreset(ctx context.Context, req K8sScopedPolicyGrantPresetRequest) (*K8sScopedPolicyGrantPresetResponse, error) {
	preset := strings.TrimSpace(req.Preset)
	switch preset {
	case string(PresetK8sReadonly), string(PresetK8sReadonlyExec), string(PresetK8sAdmin):
	default:
		return nil, constants.ErrBadRequestWithMsg("preset 须为 readonly、readonly_exec 或 admin")
	}
	if s.accessRepo == nil {
		return nil, constants.ErrInternal
	}

	kind := strings.TrimSpace(strings.ToLower(req.PrincipalKind))
	switch {
	case kind == "" && req.RoleID > 0:
		kind = model.K8sPrincipalRole
	case kind == "" && req.UserID > 0:
		kind = model.K8sPrincipalUser
	case kind == "" && req.GroupID > 0:
		kind = model.K8sPrincipalGroup
	}

	var principalRef string
	switch kind {
	case model.K8sPrincipalRole:
		if req.RoleID == 0 {
			return nil, constants.ErrBadRequestWithMsg("role_id 必填（principal_kind=role）")
		}
		if s.roleRepo == nil {
			return nil, constants.ErrInternal
		}
		role, err := s.roleRepo.GetByID(ctx, req.RoleID)
		if err != nil {
			return nil, err
		}
		principalRef = strings.TrimSpace(role.Code)
		if principalRef == "" {
			return nil, constants.ErrBadRequestWithMsg("角色编码为空")
		}
	case model.K8sPrincipalUser:
		if req.UserID == 0 {
			return nil, constants.ErrBadRequestWithMsg("user_id 必填（principal_kind=user）")
		}
		principalRef = k8sauth.UserRefString(req.UserID)
	case model.K8sPrincipalGroup:
		if req.GroupID == 0 {
			return nil, constants.ErrBadRequestWithMsg("group_id 必填（principal_kind=group）")
		}
		if s.userGroupRepo == nil {
			return nil, constants.ErrInternal
		}
		g, err := s.userGroupRepo.GetByID(ctx, req.GroupID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, constants.ErrBadRequestWithMsg("用户组不存在")
			}
			return nil, err
		}
		principalRef = strings.TrimSpace(g.Code)
		if principalRef == "" {
			return nil, constants.ErrBadRequestWithMsg("用户组编码为空")
		}
	default:
		return nil, constants.ErrBadRequestWithMsg("principal_kind 须为 role、user 或 group，或提供 role_id/user_id/group_id")
	}

	clusterIDs := req.ClusterIDs
	if len(clusterIDs) == 0 {
		clusterIDs = []uint{0}
	}

	added, skipped := 0, 0
	for _, cid := range clusterIDs {
		preList, _ := s.accessRepo.ListByPrincipal(ctx, kind, principalRef)
		had := false
		for _, g := range preList {
			if g.ClusterID == cid {
				had = true
				break
			}
		}
		it := &model.K8sClusterAccessGrant{
			PrincipalKind: kind,
			PrincipalRef:  principalRef,
			ClusterID:     cid,
			Preset:        preset,
		}
		if err := s.accessRepo.Upsert(ctx, it); err != nil {
			return nil, err
		}
		if had {
			skipped++
		} else {
			added++
		}
	}

	denyAdded, denySkipped := 0, 0
	if s.nsDenyRepo != nil && len(req.DenyNamespaces) > 0 {
		da, ds, err := syncDenyNamespaces(ctx, s.nsDenyRepo, kind, principalRef, clusterIDs, req.DenyNamespaces)
		if err != nil {
			return nil, err
		}
		denyAdded, denySkipped = da, ds
	}

	allowAdded, allowSkipped := 0, 0
	if s.nsAllowRepo != nil && len(req.AllowNamespaces) > 0 {
		aa, as, err := syncAllowNamespaces(ctx, s.nsAllowRepo, kind, principalRef, clusterIDs, req.AllowNamespaces)
		if err != nil {
			return nil, err
		}
		allowAdded, allowSkipped = aa, as
	}

	return &K8sScopedPolicyGrantPresetResponse{
		Added:             added,
		Skipped:           skipped,
		DenyRulesAdded:    denyAdded,
		DenyRulesSkipped:  denySkipped,
		AllowRulesAdded:   allowAdded,
		AllowRulesSkipped: allowSkipped,
	}, nil
}

func syncDenyNamespaces(ctx context.Context, nsDenyRepo *repository.K8sNamespaceDenyRepository, principalKind, principalRef string, clusterIDs []uint, denyNS []string) (added, skipped int, err error) {
	k := strings.TrimSpace(strings.ToLower(principalKind))
	ref := strings.TrimSpace(principalRef)
	if k == "" || ref == "" {
		return 0, 0, nil
	}
	hasWildCluster := false
	concreteClusters := make([]uint, 0, len(clusterIDs))
	for _, cid := range clusterIDs {
		if cid == 0 {
			hasWildCluster = true
			continue
		}
		concreteClusters = append(concreteClusters, cid)
	}
	if hasWildCluster || len(concreteClusters) == 0 {
		return 0, len(denyNS), nil
	}
	for _, cid := range concreteClusters {
		for _, raw := range denyNS {
			ns := strings.TrimSpace(raw)
			if ns == "" || ns == "*" || ns == "_cluster" {
				skipped++
				continue
			}
			it := &model.K8sNamespaceDenyRule{
				PrincipalKind: k,
				PrincipalRef:  ref,
				ClusterID:     cid,
				Namespace:     ns,
			}
			e := nsDenyRepo.Create(ctx, it)
			if e != nil {
				if strings.Contains(strings.ToLower(e.Error()), "duplicate") {
					skipped++
					continue
				}
				return added, skipped, e
			}
			added++
		}
	}
	return added, skipped, nil
}

func syncAllowNamespaces(ctx context.Context, nsAllowRepo *repository.K8sNamespaceAllowRepository, principalKind, principalRef string, clusterIDs []uint, allowNS []string) (added, skipped int, err error) {
	k := strings.TrimSpace(strings.ToLower(principalKind))
	ref := strings.TrimSpace(principalRef)
	if k == "" || ref == "" {
		return 0, 0, nil
	}
	hasWildCluster := false
	concreteClusters := make([]uint, 0, len(clusterIDs))
	for _, cid := range clusterIDs {
		if cid == 0 {
			hasWildCluster = true
			continue
		}
		concreteClusters = append(concreteClusters, cid)
	}
	if hasWildCluster || len(concreteClusters) == 0 {
		return 0, len(allowNS), nil
	}
	for _, cid := range concreteClusters {
		for _, raw := range allowNS {
			ns := strings.TrimSpace(raw)
			if ns == "" || ns == "*" || ns == "_cluster" {
				skipped++
				continue
			}
			it := &model.K8sNamespaceAllowRule{
				PrincipalKind: k,
				PrincipalRef:  ref,
				ClusterID:     cid,
				Namespace:     ns,
			}
			e := nsAllowRepo.Create(ctx, it)
			if e != nil {
				if strings.Contains(strings.ToLower(e.Error()), "duplicate") {
					skipped++
					continue
				}
				return added, skipped, e
			}
			added++
		}
	}
	return added, skipped, nil
}

// ListClusterGrants 按 role_id / user_id / group_id 之一列出档位（仅处理第一个非零参数，优先级 role > user > group）。
func (s *K8sScopedPolicyService) ListClusterGrants(ctx context.Context, roleID, userID, groupID uint) ([]K8sClusterAccessItem, error) {
	if s.accessRepo == nil {
		return []K8sClusterAccessItem{}, nil
	}
	var kind, ref string
	switch {
	case roleID > 0:
		if s.roleRepo == nil {
			return nil, constants.ErrInternal
		}
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return nil, err
		}
		kind, ref = model.K8sPrincipalRole, strings.TrimSpace(role.Code)
	case userID > 0:
		kind, ref = model.K8sPrincipalUser, k8sauth.UserRefString(userID)
	case groupID > 0:
		if s.userGroupRepo == nil {
			return nil, constants.ErrInternal
		}
		g, err := s.userGroupRepo.GetByID(ctx, groupID)
		if err != nil {
			return nil, err
		}
		kind, ref = model.K8sPrincipalGroup, strings.TrimSpace(g.Code)
	default:
		return []K8sClusterAccessItem{}, nil
	}
	if ref == "" {
		return []K8sClusterAccessItem{}, nil
	}
	list, err := s.accessRepo.ListByPrincipal(ctx, kind, ref)
	if err != nil {
		return nil, err
	}
	out := make([]K8sClusterAccessItem, 0, len(list))
	for _, g := range list {
		out = append(out, k8sClusterAccessItemFromGrant(g))
	}
	return out, nil
}

// ListByRole 列出角色在 DB 中的集群档位（兼容旧名）。
func (s *K8sScopedPolicyService) ListByRole(ctx context.Context, roleID uint) ([]K8sClusterAccessItem, error) {
	return s.ListClusterGrants(ctx, roleID, 0, 0)
}

// DeleteClusterGrant 删除一条集群档位。
func (s *K8sScopedPolicyService) DeleteClusterGrant(ctx context.Context, id uint) error {
	if s.accessRepo == nil {
		return constants.ErrInternal
	}
	return s.accessRepo.DeleteByID(ctx, id)
}
