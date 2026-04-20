package handler

import (
	"context"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type AdminHandler struct {
	rdb *redis.Client
}

// NewAdminHandler 创建相关逻辑。
func NewAdminHandler(rdb *redis.Client) *AdminHandler {
	return &AdminHandler{rdb: rdb}
}

func isSuperAdmin(u *auth.CurrentUser) bool {
	if u == nil {
		return false
	}
	for _, rc := range u.RoleCodes {
		if strings.TrimSpace(rc) == "super-admin" {
			return true
		}
	}
	return false
}

// ListBannedIPs lists current banned IPs (key TTL in seconds)
func (h *AdminHandler) ListBannedIPs(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || !isSuperAdmin(user) {
		response.Error(c, apperror.Forbidden("无访问权限"))
		return
	}
	ctx := c.Request.Context()
	keys, err := h.rdb.Keys(ctx, "ban:ip:*").Result()
	if err != nil {
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	result := make([]gin.H, 0, len(keys))
	for _, k := range keys {
		ttl, _ := h.rdb.TTL(ctx, k).Result()
		ip := strings.TrimPrefix(k, "ban:ip:")
		result = append(result, gin.H{"ip": ip, "ttl_seconds": int(ttl.Seconds())})
	}
	response.Success(c, gin.H{"list": result})
}

type unbanRequest struct {
	IP string `json:"ip" binding:"required,ipv4|ipv6"`
}

// UnbanIP removes a ban key
func (h *AdminHandler) UnbanIP(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || !isSuperAdmin(user) {
		response.Error(c, apperror.Forbidden("无访问权限"))
		return
	}
	handleJSONOK(c, gin.H{"message": "unbanned"}, func(ctx context.Context, req unbanRequest) error {
		key := store.BanIPKey(req.IP)
		if err := h.rdb.Del(ctx, key).Err(); err != nil {
			return apperror.Internal(err.Error())
		}
		return nil
	})
}
