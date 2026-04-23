package middleware

import (
	"strings"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func Authorize(enforcer *casbin.SyncedEnforcer, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("未登录"))
			c.Abort()
			return
		}
		// super-admin 内置放行，避免新接口尚未 seed policy 时出现“无访问权限”
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
			logger.Error.Error("casbin authorize failed", "error", err, "path", path, "method", c.Request.Method)
			response.Error(c, apperror.Internal("权限校验失败"))
			c.Abort()
			return
		}
		if !allowed {
			// 兼容策略：对于只读类接口，若已配置 K8s 三元策略，则不强制再配置一份 API 级 GET 权限。
			// 这样可避免“角色已下发三元策略，但页面仍因 /clusters、/pods GET 被拦截”的体验问题。
			if allowReadByK8sScopedPolicy(enforcer, service.UserSubject(user.ID), path, c.Request.Method) {
				c.Next()
				return
			}
			response.Error(c, apperror.Forbidden("无访问权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

func allowReadByK8sScopedPolicy(enforcer *casbin.SyncedEnforcer, subject, path, method string) bool {
	if strings.ToUpper(strings.TrimSpace(method)) != "GET" {
		return false
	}
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return false
	}
	// 菜单树是登录后基础读取能力，否则普通角色前端会先白屏再报权限不足。
	if normalizedPath == "/api/v1/menus/tree" {
		return true
	}
	// 非 K8s 资源读取，不走三元兜底。
	if !isK8sReadPath(normalizedPath) {
		return false
	}

	perms, err := enforcer.GetImplicitPermissionsForUser(subject)
	if err != nil || len(perms) == 0 {
		return false
	}
	// clusters 列表没有 cluster_id 维度，判定为“只要有任意 k8s 三元策略即可查看集群列表”。
	if normalizedPath == "/api/v1/clusters" {
		for _, p := range perms {
			if len(p) >= 3 && strings.HasPrefix(strings.TrimSpace(p[1]), "k8s:cluster:") {
				return true
			}
		}
		return false
	}
	targetSuffix := ":" + normalizedPath
	for _, p := range perms {
		if len(p) < 3 {
			continue
		}
		obj := strings.TrimSpace(p[1])
		if !strings.HasPrefix(obj, "k8s:cluster:") {
			continue
		}
		if strings.HasSuffix(obj, targetSuffix) {
			return true
		}
	}
	return false
}

func isK8sReadPath(path string) bool {
	p := strings.TrimSpace(path)
	k8sPrefixes := []string{
		"/api/v1/clusters",
		"/api/v1/pods",
		"/api/v1/namespaces",
		"/api/v1/nodes",
		"/api/v1/deployments",
		"/api/v1/statefulsets",
		"/api/v1/daemonsets",
		"/api/v1/cronjobs",
		"/api/v1/jobs",
		"/api/v1/configmaps",
		"/api/v1/secrets",
		"/api/v1/k8s-services",
		"/api/v1/persistentvolumes",
		"/api/v1/persistentvolumeclaims",
		"/api/v1/storageclasses",
		"/api/v1/ingresses",
		"/api/v1/events",
		"/api/v1/crds",
		"/api/v1/crs",
		"/api/v1/rbac",
	}
	for _, prefix := range k8sPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
