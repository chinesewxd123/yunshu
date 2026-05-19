package middleware

import (
	"strings"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func Authorize(enforcer *casbin.SyncedEnforcer, logger *logx.Logger, k8sAccessRepo *repository.K8sClusterAccessRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, constants.ErrNotLoggedIn)
			c.Abort()
			return
		}
		for _, rc := range user.RoleCodes {
			if rc == "super-admin" {
				c.Next()
				return
			}
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		allowed, err := enforcer.Enforce(service.UserSubject(user.ID), path, c.Request.Method)
		if err != nil {
			httpLog("http.authorize").Error("casbin authorize failed", "error", err, "path", path, "method", c.Request.Method)
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}
		if !allowed {
			if allowReadByK8sClusterGrant(c, k8sAccessRepo, user, path, c.Request.Method) {
				c.Next()
				return
			}
			response.Error(c, constants.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// allowReadByK8sClusterGrant：未配置 API 级 GET 时，若角色在 DB 中有 K8s 集群档位且满足只读场景则放行（与旧版 Casbin 三元兜底等价）。
func allowReadByK8sClusterGrant(c *gin.Context, accessRepo *repository.K8sClusterAccessRepository, user *auth.CurrentUser, path, method string) bool {
	if accessRepo == nil || user == nil {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(method)) != "GET" {
		return false
	}
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return false
	}
	if normalizedPath == "/api/v1/menus/tree" {
		return true
	}
	if !service.IsK8sReadAPIPath(normalizedPath) {
		return false
	}
	ctx := c.Request.Context()
	pack := k8sauth.PackFromCurrentUser(user)
	if normalizedPath == "/api/v1/clusters" {
		return accessRepo.HasAnyK8sGrant(ctx, pack)
	}
	clusterID, _ := extractClusterNamespaceFromRequest(c)
	if clusterID == 0 {
		return false
	}
	rank := accessRepo.EffectiveTier(ctx, pack, clusterID)
	return rank >= service.K8sAccessRankReadonly
}
