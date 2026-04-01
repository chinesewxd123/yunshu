package handler

import (
	"time"

	"go-permission-system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	name      string
	env       string
	startTime time.Time
}

func NewSystemHandler(name, env string) *SystemHandler {
	return &SystemHandler{name: name, env: env, startTime: time.Now()}
}

func (h *SystemHandler) Health(c *gin.Context) {
	uptime := int(time.Since(h.startTime).Seconds())
	response.Success(c, gin.H{
		"status":  "ok",
		"version": h.name + "@" + h.env,
		"uptime":  uptime,
	})
}
