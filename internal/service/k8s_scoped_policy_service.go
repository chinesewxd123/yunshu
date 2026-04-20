package service

import (
	"context"
	"strconv"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/repository"

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

type K8sScopedPolicyService struct {
	roleRepo *repository.RoleRepository
	permRepo *repository.PermissionRepository
	enforcer *casbin.SyncedEnforcer
}

// NewK8sScopedPolicyService 创建相关逻辑。
func NewK8sScopedPolicyService(
	roleRepo *repository.RoleRepository,
	permRepo *repository.PermissionRepository,
	enforcer *casbin.SyncedEnforcer,
) *K8sScopedPolicyService {
	return &K8sScopedPolicyService{
		roleRepo: roleRepo,
		permRepo: permRepo,
		enforcer: enforcer,
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
		return nil, apperror.BadRequest("actions 与 paths 不能为空")
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

// ListByRole 查询列表相关的业务逻辑。
func (s *K8sScopedPolicyService) ListByRole(ctx context.Context, roleID uint) ([]K8sScopedPolicyItem, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	roleCode := role.Code

	policies, err := s.enforcer.GetFilteredPolicy(0, roleCode)
	if err != nil {
		return nil, err
	}

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
