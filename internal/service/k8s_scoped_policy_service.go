package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
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
	roleRepo      *repository.RoleRepository
	permRepo      *repository.PermissionRepository
	accessRepo    *repository.K8sClusterAccessRepository
	nsDenyRepo    *repository.K8sNamespaceDenyRepository
	nsAllowRepo   *repository.K8sNamespaceAllowRepository
	userGroupRepo *repository.UserGroupRepository
	userRepo      *repository.UserRepository
	clusterRepo   *repository.K8sClusterRepository
}

// K8sAuthMatrixRow 集群管理「已授权」矩阵行（对齐 k8m：按用户展开角色/组授权）。
type K8sAuthMatrixRow struct {
	RowKey          string `json:"row_key"`
	GrantID         uint   `json:"grant_id"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname,omitempty"`
	PrincipalKind   string `json:"principal_kind"`
	PrincipalRef    string `json:"principal_ref"`
	PrincipalShow   string `json:"principal_show"`
	ClusterID       uint   `json:"cluster_id"`
	ClusterName     string `json:"cluster_name"`
	GrantScopeAll   bool   `json:"grant_scope_all"`
	Preset          string `json:"preset"`
	PresetLabel     string `json:"preset_label"`
	AllowNamespaces string `json:"allow_namespaces"`
	Via             string `json:"via"`
}

// K8sUserClusterAuthRow 用户管理「已授权集群」行。
type K8sUserClusterAuthRow struct {
	RowKey          string `json:"row_key"`
	GrantID         uint   `json:"grant_id"`
	Username        string `json:"username"`
	ClusterID       uint   `json:"cluster_id"`
	ClusterName     string `json:"cluster_name"`
	GrantScopeAll   bool   `json:"grant_scope_all"`
	Preset          string `json:"preset"`
	PresetLabel     string `json:"preset_label"`
	AllowNamespaces string `json:"allow_namespaces"`
	Via             string `json:"via"`
}

// NewK8sScopedPolicyService 创建 K8s 集群档位服务（不写 Casbin k8s: 策略）。
func NewK8sScopedPolicyService(
	roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	accessRepo *repository.K8sClusterAccessRepository,
	nsDenyRepo *repository.K8sNamespaceDenyRepository,
	nsAllowRepo *repository.K8sNamespaceAllowRepository,
	userGroupRepo *repository.UserGroupRepository,
	userRepo *repository.UserRepository,
	clusterRepo *repository.K8sClusterRepository,
) *K8sScopedPolicyService {
	return &K8sScopedPolicyService{
		roleRepo:      roleRepo,
		permRepo:      permRepo,
		accessRepo:    accessRepo,
		nsDenyRepo:    nsDenyRepo,
		nsAllowRepo:   nsAllowRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		clusterRepo:   clusterRepo,
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
			return nil, svcerr.Pass("k8s.policy", "GrantPreset", err)
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
			return nil, svcerr.Pass("k8s.policy", "GrantPreset", err)
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
			return nil, svcerr.Pass("k8s.policy", "GrantPreset", err)
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
			return nil, svcerr.Pass("k8s.policy", "GrantPreset", err)
		}
		denyAdded, denySkipped = da, ds
	}

	allowAdded, allowSkipped := 0, 0
	if s.nsAllowRepo != nil && len(req.AllowNamespaces) > 0 {
		aa, as, err := syncAllowNamespaces(ctx, s.nsAllowRepo, kind, principalRef, clusterIDs, req.AllowNamespaces)
		if err != nil {
			return nil, svcerr.Pass("k8s.policy", "GrantPreset", err)
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
			return nil, svcerr.Pass("k8s.policy", "ListClusterGrants", err)
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
			return nil, svcerr.Pass("k8s.policy", "ListClusterGrants", err)
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
		return nil, svcerr.Pass("k8s.policy", "ListClusterGrants", err)
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

// DeleteClusterGrantsBatch 批量删除集群档位（去重 id）。
func (s *K8sScopedPolicyService) DeleteClusterGrantsBatch(ctx context.Context, ids []uint) (deleted int, err error) {
	if s.accessRepo == nil {
		return 0, constants.ErrInternal
	}
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if e := s.accessRepo.DeleteByID(ctx, id); e != nil {
			return deleted, e
		}
		deleted++
	}
	return deleted, nil
}

func presetLabelCN(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "admin":
		return "集群管理员"
	case "readonly_exec":
		return "Exec 权限"
	case "readonly":
		return "集群只读"
	default:
		return strings.TrimSpace(p)
	}
}

func viaForPrincipalKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case model.K8sPrincipalRole:
		return "经责任域角色"
	case model.K8sPrincipalGroup:
		return "经用户组"
	case model.K8sPrincipalUser:
		return "用户直授"
	default:
		return ""
	}
}

func parseUserRefUint(ref string) uint {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0
	}
	n, err := strconv.ParseUint(ref, 10, 32)
	if err != nil {
		return 0
	}
	return uint(n)
}

func (s *K8sScopedPolicyService) joinAllowNamespaces(ctx context.Context, kind, ref string, clusterID uint) string {
	if s.nsAllowRepo == nil {
		return ""
	}
	ns, err := s.nsAllowRepo.DistinctNamespacesForPrincipalCluster(ctx, kind, ref, clusterID)
	if err != nil || len(ns) == 0 {
		return ""
	}
	return strings.Join(ns, ", ")
}

func (s *K8sScopedPolicyService) formatPrincipalShow(ctx context.Context, kind, ref string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	ref = strings.TrimSpace(ref)
	switch k {
	case model.K8sPrincipalRole:
		return "角色:" + ref
	case model.K8sPrincipalGroup:
		if s.userGroupRepo == nil {
			return "组:" + ref
		}
		g, err := s.userGroupRepo.GetByCode(ctx, ref)
		if err == nil && g != nil {
			return fmt.Sprintf("组:%s（%s）", strings.TrimSpace(g.Name), ref)
		}
		return "组:" + ref
	case model.K8sPrincipalUser:
		uid := parseUserRefUint(ref)
		if uid > 0 && s.userRepo != nil {
			u, err := s.userRepo.GetByID(ctx, uid)
			if err == nil && u != nil {
				return fmt.Sprintf("用户:%s", strings.TrimSpace(u.Username))
			}
		}
		return "用户ID:" + ref
	default:
		return ref
	}
}

// ListClusterAuthMatrix 指定集群下已授权主体按用户展开（含 cluster_id=0 的全局档）。
func (s *K8sScopedPolicyService) ListClusterAuthMatrix(ctx context.Context, clusterID uint) ([]K8sAuthMatrixRow, error) {
	if clusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("cluster_id 必填")
	}
	if s.accessRepo == nil || s.clusterRepo == nil || s.userRepo == nil {
		return nil, constants.ErrInternal
	}
	clu, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
	}
	viewName := strings.TrimSpace(clu.Name)
	grants, err := s.accessRepo.ListGrantsApplyingToCluster(ctx, clusterID)
	if err != nil {
		return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
	}
	var out []K8sAuthMatrixRow
	appendRow := func(g model.K8sClusterAccessGrant, username, nickname string, rowSuffix string) {
		allow := s.joinAllowNamespaces(ctx, g.PrincipalKind, g.PrincipalRef, clusterID)
		rk := fmt.Sprintf("g%d-%s", g.ID, rowSuffix)
		out = append(out, K8sAuthMatrixRow{
			RowKey:          rk,
			GrantID:         g.ID,
			Username:        username,
			Nickname:        nickname,
			PrincipalKind:   g.PrincipalKind,
			PrincipalRef:    g.PrincipalRef,
			PrincipalShow:   s.formatPrincipalShow(ctx, g.PrincipalKind, g.PrincipalRef),
			ClusterID:       clusterID,
			ClusterName:     viewName,
			GrantScopeAll:   g.ClusterID == 0,
			Preset:          g.Preset,
			PresetLabel:     presetLabelCN(g.Preset),
			AllowNamespaces: allow,
			Via:             viaForPrincipalKind(g.PrincipalKind),
		})
	}

	for _, g := range grants {
		k := strings.ToLower(strings.TrimSpace(g.PrincipalKind))
		switch k {
		case model.K8sPrincipalUser:
			uid := parseUserRefUint(g.PrincipalRef)
			uname, nick := "-", ""
			if uid > 0 {
				u, err := s.userRepo.GetByID(ctx, uid)
				if err == nil && u != nil {
					uname = strings.TrimSpace(u.Username)
					nick = strings.TrimSpace(u.Nickname)
				}
			}
			appendRow(g, uname, nick, fmt.Sprintf("u%d", uid))
		case model.K8sPrincipalRole:
			ids, err := s.userRepo.ListUserIDsByRoleCode(ctx, g.PrincipalRef)
			if err != nil {
				return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
			}
			if len(ids) == 0 {
				appendRow(g, "-", "(当前无用户绑定该角色)", "role-empty")
				continue
			}
			users, err := s.userRepo.ListByIDs(ctx, ids)
			if err != nil {
				return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
			}
			byID := map[uint]model.User{}
			for _, u := range users {
				byID[u.ID] = u
			}
			for _, uid := range ids {
				u := byID[uid]
				appendRow(g, strings.TrimSpace(u.Username), strings.TrimSpace(u.Nickname), fmt.Sprintf("r%d", uid))
			}
		case model.K8sPrincipalGroup:
			if s.userGroupRepo == nil {
				appendRow(g, "-", "(无法解析用户组)", "grp-err")
				continue
			}
			grp, err := s.userGroupRepo.GetByCode(ctx, g.PrincipalRef)
			if err != nil || grp == nil {
				appendRow(g, "-", "(用户组不存在)", "grp-miss")
				continue
			}
			mids, err := s.userGroupRepo.ListMemberUserIDs(ctx, grp.ID)
			if err != nil {
				return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
			}
			if len(mids) == 0 {
				appendRow(g, "-", "(组内暂无成员)", "grp-empty")
				continue
			}
			users, err := s.userRepo.ListByIDs(ctx, mids)
			if err != nil {
				return nil, svcerr.Pass("k8s.policy", "ListClusterAuthMatrix", err)
			}
			byID := map[uint]model.User{}
			for _, u := range users {
				byID[u.ID] = u
			}
			for _, uid := range mids {
				u := byID[uid]
				appendRow(g, strings.TrimSpace(u.Username), strings.TrimSpace(u.Nickname), fmt.Sprintf("g%d", uid))
			}
		default:
			appendRow(g, "-", "", "x")
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Username != out[j].Username {
			return out[i].Username < out[j].Username
		}
		return out[i].GrantID < out[j].GrantID
	})
	return out, nil
}

type userAuthSource struct {
	kind, ref, via string
}

// ListUserClusterAuth 汇总某用户通过直授 / 角色 / 用户组获得的集群档位（展开「全部集群」档）。
func (s *K8sScopedPolicyService) ListUserClusterAuth(ctx context.Context, userID uint) ([]K8sUserClusterAuthRow, error) {
	if userID == 0 {
		return nil, constants.ErrBadRequestWithMsg("user_id 必填")
	}
	if s.userRepo == nil || s.accessRepo == nil || s.clusterRepo == nil {
		return nil, constants.ErrInternal
	}
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, svcerr.Pass("k8s.policy", "ListUserClusterAuth", err)
	}
	clusters, err := s.clusterRepo.ListAllBrief(ctx)
	if err != nil {
		return nil, svcerr.Pass("k8s.policy", "ListUserClusterAuth", err)
	}
	id2name := map[uint]string{}
	for _, c := range clusters {
		id2name[c.ID] = strings.TrimSpace(c.Name)
	}

	sources := []userAuthSource{
		{model.K8sPrincipalUser, k8sauth.UserRefString(userID), "用户直授"},
	}
	for _, role := range u.Roles {
		rc := strings.TrimSpace(role.Code)
		if rc == "" {
			continue
		}
		sources = append(sources, userAuthSource{model.K8sPrincipalRole, rc, "责任域角色 " + strings.TrimSpace(role.Name)})
	}
	for _, g := range u.Groups {
		gc := strings.TrimSpace(g.Code)
		if gc == "" {
			continue
		}
		sources = append(sources, userAuthSource{model.K8sPrincipalGroup, gc, "用户组 " + strings.TrimSpace(g.Name)})
	}

	var out []K8sUserClusterAuthRow
	uname := strings.TrimSpace(u.Username)

	for _, sc := range sources {
		grants, err := s.accessRepo.ListByPrincipal(ctx, sc.kind, sc.ref)
		if err != nil {
			return nil, svcerr.Pass("k8s.policy", "ListUserClusterAuth", err)
		}
		for _, g := range grants {
			if g.ClusterID == 0 {
				allow := s.joinAllowNamespaces(ctx, sc.kind, sc.ref, 0)
				out = append(out, K8sUserClusterAuthRow{
					RowKey:          fmt.Sprintf("%d-0-%s-%s", g.ID, sc.kind, sc.ref),
					GrantID:         g.ID,
					Username:        uname,
					ClusterID:       0,
					ClusterName:     "全部集群",
					GrantScopeAll:   true,
					Preset:          g.Preset,
					PresetLabel:     presetLabelCN(g.Preset),
					AllowNamespaces: allow,
					Via:             sc.via,
				})
				continue
			}
			cid := g.ClusterID
			cname := id2name[cid]
			if cname == "" {
				cname = fmt.Sprintf("集群#%d", cid)
			}
			allow := s.joinAllowNamespaces(ctx, sc.kind, sc.ref, cid)
			out = append(out, K8sUserClusterAuthRow{
				RowKey:          fmt.Sprintf("%d-%d-%s-%s", g.ID, cid, sc.kind, sc.ref),
				GrantID:         g.ID,
				Username:        uname,
				ClusterID:       cid,
				ClusterName:     cname,
				GrantScopeAll:   false,
				Preset:          g.Preset,
				PresetLabel:     presetLabelCN(g.Preset),
				AllowNamespaces: allow,
				Via:             sc.via,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ClusterName != out[j].ClusterName {
			return out[i].ClusterName < out[j].ClusterName
		}
		if out[i].Via != out[j].Via {
			return out[i].Via < out[j].Via
		}
		return out[i].GrantID < out[j].GrantID
	})
	return out, nil
}
