package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type k8sScopeCatalogCache struct {
	repo        *repository.PermissionRepository
	ttl         time.Duration
	mu          sync.RWMutex
	loadedAt    time.Time
	actionByKey map[string]string
	scopedKeys  map[string]bool
}

func newK8sScopeCatalogCache(repo *repository.PermissionRepository) *k8sScopeCatalogCache {
	return &k8sScopeCatalogCache{
		repo: repo,
		ttl:  60 * time.Second,
	}
}

func (c *k8sScopeCatalogCache) get(routePath, method string) (string, bool) {
	if c == nil {
		return "", false
	}
	now := time.Now()
	c.mu.RLock()
	ready := !c.loadedAt.IsZero() && now.Sub(c.loadedAt) < c.ttl
	actionByKey := c.actionByKey
	scopedKeys := c.scopedKeys
	c.mu.RUnlock()
	if !ready {
		c.refresh()
		c.mu.RLock()
		actionByKey = c.actionByKey
		scopedKeys = c.scopedKeys
		c.mu.RUnlock()
	}
	key := service.K8sScopeRouteKey(routePath, method)
	if scopedKeys == nil || !scopedKeys[key] {
		return "", false
	}
	return strings.TrimSpace(actionByKey[key]), true
}

func (c *k8sScopeCatalogCache) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.repo == nil {
		c.actionByKey = map[string]string{}
		c.scopedKeys = map[string]bool{}
		c.loadedAt = time.Now()
		return
	}
	perms, err := c.repo.ListAll(context.Background())
	if err != nil {
		return
	}
	actionByKey, scopedKeys := service.BuildK8sScopeMappings(perms)
	c.actionByKey = actionByKey
	c.scopedKeys = scopedKeys
	c.loadedAt = time.Now()
}

// K8sScopeAuthorize:
// 1) 仅对纳入三元目录的 K8s 接口启用（来自 permissions 动态构建）
// 2) 支持 cluster + namespace + action 三元权限
// 3) 兼容旧权限：当未配置任何三元策略时默认放行
func K8sScopeAuthorize(
	enforcer *casbin.SyncedEnforcer,
	logger *logx.Logger,
	permRepo *repository.PermissionRepository,
) gin.HandlerFunc {
	catalog := newK8sScopeCatalogCache(permRepo)
	return func(c *gin.Context) {
		routePath := c.FullPath()
		if strings.TrimSpace(routePath) == "" {
			routePath = c.Request.URL.Path
		}
		actionCode, tracked := catalog.get(routePath, c.Request.Method)
		if !tracked {
			c.Next()
			return
		}
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("未登录"))
			c.Abort()
			return
		}
		for _, rc := range user.RoleCodes {
			if strings.TrimSpace(rc) == "super-admin" {
				c.Next()
				return
			}
		}

		clusterID, namespace := extractClusterNamespaceFromRequest(c)
		if clusterID == 0 {
			// 无 cluster_id 的旧接口不拦截
			c.Next()
			return
		}
		if strings.TrimSpace(namespace) == "" {
			namespace = "_cluster"
		}

		path := routePath
		method := c.Request.Method
		actionCandidates := []string{}
		if strings.TrimSpace(actionCode) != "" {
			actionCandidates = append(actionCandidates, actionCode)
		}
		// 兼容旧策略：仍允许使用 HTTP method 作为 action
		actionCandidates = append(actionCandidates, method)
		subject := service.UserSubject(user.ID)

		resources := []string{
			buildK8sScopeResource(clusterID, namespace, path),
			buildK8sScopeResource(clusterID, "*", path),
			buildK8sScopeResource(0, namespace, path),
			buildK8sScopeResource(0, "*", path),
		}

		hasScopedPolicy := false
		allowed := false
		for _, res := range resources {
			for _, act := range actionCandidates {
				policies := enforcer.GetFilteredPolicy(1, res, act)
				if len(policies) == 0 {
					continue
				}
				hasScopedPolicy = true
				ok, err := enforcer.Enforce(subject, res, act)
				if err != nil {
					logger.Error.Error("enforce scoped policy failed", "error", err, "resource", res, "action", act)
					response.Error(c, apperror.Internal("权限校验失败"))
					c.Abort()
					return
				}
				if ok {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}

		if hasScopedPolicy && !allowed {
			response.Error(c, apperror.Forbidden("无该集群/命名空间操作权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

func buildK8sScopeResource(clusterID uint, namespace, path string) string {
	clusterPart := "*"
	if clusterID > 0 {
		clusterPart = strconv.FormatUint(uint64(clusterID), 10)
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_cluster"
	}
	return "k8s:cluster:" + clusterPart + ":ns:" + ns + ":" + strings.TrimSpace(path)
}

func extractClusterNamespaceFromRequest(c *gin.Context) (uint, string) {
	clusterID := parseUint(strings.TrimSpace(c.Query("cluster_id")))
	namespace := strings.TrimSpace(c.Query("namespace"))

	if clusterID == 0 {
		clusterID = parseUint(strings.TrimSpace(c.Param("id")))
	}

	if (clusterID == 0 || namespace == "") && c.Request.Body != nil {
		raw, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		if len(raw) > 0 {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err == nil {
				if clusterID == 0 {
					clusterID = toUint(m["cluster_id"])
					if clusterID == 0 {
						clusterID = toUint(m["clusterId"])
					}
				}
				if namespace == "" {
					if s, ok := m["namespace"].(string); ok {
						namespace = strings.TrimSpace(s)
					}
				}
			}
		}
	}

	if clusterID == 0 {
		clusterID = parseUint(strings.TrimSpace(c.PostForm("cluster_id")))
	}
	if namespace == "" {
		namespace = strings.TrimSpace(c.PostForm("namespace"))
	}

	return clusterID, namespace
}

func parseUint(v string) uint {
	if strings.TrimSpace(v) == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func toUint(v any) uint {
	switch t := v.(type) {
	case float64:
		return uint(t)
	case int:
		return uint(t)
	case int64:
		return uint(t)
	case uint:
		return t
	case string:
		return parseUint(t)
	default:
		return 0
	}
}
