package service

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/repository"

	"github.com/casbin/casbin/v2"
)

type K8sActionItem struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type K8sScopedPolicyGrantRequest struct {
	RoleID     uint     `json:"role_id" binding:"required"`
	ClusterIDs []uint   `json:"cluster_ids"`
	Namespaces []string `json:"namespaces"`
	Actions    []string `json:"actions" binding:"required"`
	Paths      []string `json:"paths" binding:"required"`
}

type K8sScopedPolicyGrantResponse struct {
	Added    int      `json:"added"`
	Skipped  int      `json:"skipped"`
	Policies []string `json:"policies"`
}

type K8sScopedPolicyItem struct {
	RoleCode  string `json:"role_code"`
	ClusterID string `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Path      string `json:"path"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
}

// K8sScopedPolicyGrantPresetRequest 按 k8m 风格档位一键下发三元策略，并可同步命名空间黑名单规则。
type K8sScopedPolicyGrantPresetRequest struct {
	RoleID         uint     `json:"role_id" binding:"required"`
	ClusterIDs     []uint   `json:"cluster_ids"`
	Namespaces     []string `json:"namespaces"`
	Preset         string   `json:"preset" binding:"required"` // readonly | readonly_exec | admin
	DenyNamespaces []string `json:"deny_namespaces"`           // 可选；对每个已选集群写入黑名单（需明确 cluster_ids）
}

// K8sScopedPolicyGrantPresetResponse 预设下发结果。
type K8sScopedPolicyGrantPresetResponse struct {
	Added            int      `json:"added"`
	Skipped          int      `json:"skipped"`
	Policies         []string `json:"policies"`
	DenyRulesAdded   int      `json:"deny_rules_added"`
	DenyRulesSkipped int      `json:"deny_rules_skipped"`
}

type K8sScopedPolicyService struct {
	roleRepo   *repository.RoleRepository
	permRepo   *repository.PermissionRepository
	enforcer   *casbin.SyncedEnforcer
	nsDenyRepo *repository.K8sNamespaceDenyRepository
}

// NewK8sScopedPolicyService 创建相关逻辑。
func NewK8sScopedPolicyService(
	roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	enforcer *casbin.SyncedEnforcer,
	nsDenyRepo *repository.K8sNamespaceDenyRepository,
) *K8sScopedPolicyService {
	return &K8sScopedPolicyService{
		roleRepo:   roleRepo,
		permRepo:   permRepo,
		enforcer:   enforcer,
		nsDenyRepo: nsDenyRepo,
	}
}

// ActionCatalog 执行对应的业务逻辑。
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

// PathCatalog 执行对应的业务逻辑。
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

// Grant 执行对应的业务逻辑。
func (s *K8sScopedPolicyService) Grant(ctx context.Context, req K8sScopedPolicyGrantRequest) (*K8sScopedPolicyGrantResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	roleCode := role.Code

	if len(req.Actions) == 0 || len(req.Paths) == 0 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg316eb6f964a1)
	}

	namespaces := req.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{"*"}
	}

	clusterIDs := req.ClusterIDs
	if len(clusterIDs) == 0 {
		clusterIDs = []uint{0}
	}

	policies := make([][]string, 0, len(clusterIDs)*len(namespaces)*len(req.Paths)*len(req.Actions))
	flat := make([]string, 0, cap(policies))
	for _, cid := range clusterIDs {
		clusterPart := "*"
		if cid > 0 {
			clusterPart = strconv.FormatUint(uint64(cid), 10)
		}
		for _, ns := range namespaces {
			ns = strings.TrimSpace(ns)
			if ns == "" {
				ns = "*"
			}
			for _, p := range req.Paths {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				res := "k8s:cluster:" + clusterPart + ":ns:" + ns + ":" + p
				for _, act := range req.Actions {
					act = strings.TrimSpace(act)
					if act == "" {
						continue
					}
					policies = append(policies, []string{roleCode, res, act})
					flat = append(flat, roleCode+" "+res+" "+act)
				}
			}
		}
	}

	addCount := 0
	skipped := 0
	for _, pol := range policies {
		if len(pol) != 3 {
			skipped++
			continue
		}
		ok, err := s.enforcer.AddPolicy(pol[0], pol[1], pol[2])
		if err != nil {
			return nil, err
		}
		if ok {
			addCount++
		} else {
			skipped++
		}
	}
	return &K8sScopedPolicyGrantResponse{
		Added:    addCount,
		Skipped:  skipped,
		Policies: flat,
	}, nil
}

// GrantPreset 按档位批量下发 path+action 配对（非 paths×actions 笛卡尔积）。
func (s *K8sScopedPolicyService) GrantPreset(ctx context.Context, req K8sScopedPolicyGrantPresetRequest) (*K8sScopedPolicyGrantPresetResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	preset := K8sClusterAccessPreset(strings.TrimSpace(req.Preset))
	if preset != PresetK8sReadonly && preset != PresetK8sReadonlyExec && preset != PresetK8sAdmin {
		return nil, constants.ErrBadRequestWithMsg("preset 须为 readonly、readonly_exec 或 admin")
	}
	if s.permRepo == nil {
		return nil, constants.ErrInternal
	}
	perms, err := s.permRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	pairs := expandPresetTriples(perms, preset)
	if len(pairs) == 0 {
		return nil, constants.ErrBadRequestWithMsg("当前权限目录无法展开该预设，请检查 permissions 表或 seed")
	}

	namespaces := req.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{"*"}
	}
	clusterIDs := req.ClusterIDs
	if len(clusterIDs) == 0 {
		clusterIDs = []uint{0}
	}

	addCount, skipped, flat, err := s.addPairedK8sPolicies(role.Code, clusterIDs, namespaces, pairs)
	if err != nil {
		return nil, err
	}

	denyAdded, denySkipped := 0, 0
	if s.nsDenyRepo != nil && len(req.DenyNamespaces) > 0 {
		da, ds, err := s.syncDenyNamespaces(ctx, role.Code, clusterIDs, req.DenyNamespaces)
		if err != nil {
			return nil, err
		}
		denyAdded, denySkipped = da, ds
	}

	return &K8sScopedPolicyGrantPresetResponse{
		Added:            addCount,
		Skipped:          skipped,
		Policies:         flat,
		DenyRulesAdded:   denyAdded,
		DenyRulesSkipped: denySkipped,
	}, nil
}

func (s *K8sScopedPolicyService) addPairedK8sPolicies(roleCode string, clusterIDs []uint, namespaces []string, pairs []policyPathAction) (added, skipped int, flat []string, err error) {
	flat = make([]string, 0, len(clusterIDs)*len(namespaces)*len(pairs))
	for _, cid := range clusterIDs {
		clusterPart := "*"
		if cid > 0 {
			clusterPart = strconv.FormatUint(uint64(cid), 10)
		}
		for _, ns := range namespaces {
			ns = strings.TrimSpace(ns)
			if ns == "" {
				ns = "*"
			}
			for _, pair := range pairs {
				p := strings.TrimSpace(pair.path)
				act := strings.TrimSpace(pair.action)
				if p == "" || act == "" {
					skipped++
					continue
				}
				res := "k8s:cluster:" + clusterPart + ":ns:" + ns + ":" + p
				ok, e := s.enforcer.AddPolicy(roleCode, res, act)
				if e != nil {
					return added, skipped, flat, e
				}
				flat = append(flat, roleCode+" "+res+" "+act)
				if ok {
					added++
				} else {
					skipped++
				}
			}
		}
	}
	return added, skipped, flat, nil
}

func (s *K8sScopedPolicyService) syncDenyNamespaces(ctx context.Context, roleCode string, clusterIDs []uint, denyNS []string) (added, skipped int, err error) {
	rc := strings.TrimSpace(roleCode)
	if rc == "" {
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
				RoleCode:  rc,
				ClusterID: cid,
				Namespace: ns,
			}
			e := s.nsDenyRepo.Create(ctx, it)
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

// ListByRole 查询列表相关的业务逻辑。
func (s *K8sScopedPolicyService) ListByRole(ctx context.Context, roleID uint) ([]K8sScopedPolicyItem, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	roleCode := role.Code

	policies := s.enforcer.GetFilteredPolicy(0, roleCode)

	out := make([]K8sScopedPolicyItem, 0, len(policies))
	for _, p := range policies {
		if len(p) < 3 {
			continue
		}
		obj := p[1]
		act := p[2]
		if !strings.HasPrefix(obj, "k8s:cluster:") {
			continue
		}
		clusterID, namespace, path := parseK8sScopeResource(obj)
		out = append(out, K8sScopedPolicyItem{
			RoleCode:  roleCode,
			ClusterID: clusterID,
			Namespace: namespace,
			Path:      path,
			Action:    act,
			Resource:  obj,
		})
	}
	return out, nil
}

func parseK8sScopeResource(res string) (clusterID, namespace, path string) {
	// k8s:cluster:<id|*>:ns:<ns>:<path>
	s := strings.TrimSpace(res)
	if s == "" {
		return "", "", ""
	}
	parts := strings.SplitN(s, ":ns:", 2)
	if len(parts) != 2 {
		return "", "", ""
	}
	left := parts[0] // k8s:cluster:<id|*>
	right := parts[1]
	leftParts := strings.Split(left, ":")
	if len(leftParts) >= 3 {
		clusterID = leftParts[len(leftParts)-1]
	}
	nsAndPath := strings.SplitN(right, ":", 2)
	if len(nsAndPath) == 2 {
		namespace = nsAndPath[0]
		path = nsAndPath[1]
	}
	return clusterID, namespace, path
}
