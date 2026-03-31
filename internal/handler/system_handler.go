package handler

import (
	"go-permission-system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	name string
	env  string
}

func NewSystemHandler(name, env string) *SystemHandler {
	return &SystemHandler{name: name, env: env}
}

// Health godoc
// @Summary Health check
// @Description Return service name and current environment.
// @Tags System
// @Produce json
// @Success 200 {object} response.Body{data=HealthData} "success"
// @Router /api/v1/health [get]
func (h *SystemHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{
		"name": h.name,
		"env":  h.env,
	})
}
