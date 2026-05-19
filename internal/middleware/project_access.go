package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/projectaccess"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireProjectMemberAccess 校验当前用户是否为 URL 中 :id 对应项目的成员；超级管理员跳过。
// 规则：GET/HEAD 任意成员均可；非 GET 只读成员禁止；项目元数据（PUT/DELETE /projects/:id）与成员管理（/members 且非 GET）需 admin 或 owner。
func RequireProjectMemberAccess(memberRepo *repository.ProjectMemberRepository, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if memberRepo == nil {
			c.Next()
			return
		}
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, constants.ErrNotLoggedIn)
			c.Abort()
			return
		}
		if auth.IsSuperAdminRole(user.RoleCodes) {
			c.Next()
			return
		}
		idStr := strings.TrimSpace(c.Param("id"))
		if idStr == "" {
			c.Next()
			return
		}
		pv, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil || pv == 0 {
			response.Error(c, constants.ErrInvalidRequestParam)
			c.Abort()
			return
		}
		projectID := uint(pv)
		m, err := memberRepo.GetByProjectAndUser(c.Request.Context(), projectID, user.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, constants.ErrProjectMemberRequired)
				c.Abort()
				return
			}
			httpLog("http.project_access").Error("project member lookup failed", "error", err)
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}
		role := m.Role
		fullPath := c.FullPath()
		if fullPath == "" {
			fullPath = c.Request.URL.Path
		}
		method := strings.ToUpper(strings.TrimSpace(c.Request.Method))

		if isProjectAdminOnlyRoute(method, fullPath) {
			if !projectaccess.RoleAtLeast(role, "admin") {
				response.Error(c, constants.ErrProjectAdminRequired)
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if method != http.MethodGet && method != http.MethodHead {
			if projectaccess.IsReadonly(role) {
				response.Error(c, constants.ErrProjectReadonlyMember)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func isProjectAdminOnlyRoute(method, ginFullPath string) bool {
	m := strings.ToUpper(strings.TrimSpace(method))
	if m == http.MethodGet || m == http.MethodHead {
		return false
	}
	if ginFullPath == "/api/v1/projects/:id" && (m == http.MethodPut || m == http.MethodDelete) {
		return true
	}
	if strings.Contains(ginFullPath, "/api/v1/projects/:id/members") {
		return true
	}
	return false
}
