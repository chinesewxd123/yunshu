package handler

import (
	"time"

	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	name      string
	env       string
	startTime time.Time
}

// NewSystemHandler 创建相关逻辑。
func NewSystemHandler(name, env string) *SystemHandler {
	return &SystemHandler{name: name, env: env, startTime: time.Now()}
}

// Health 处理对应的 HTTP 请求并返回统一响应。
func (h *SystemHandler) Health(c *gin.Context) {
	uptime := int(time.Since(h.startTime).Seconds())
	response.Success(c, gin.H{
		"status":  "ok",
		"version": h.name + "@" + h.env,
		"uptime":  uptime,
	})
}
