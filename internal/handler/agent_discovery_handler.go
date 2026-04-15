package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type AgentDiscoveryHandler struct {
	svc *service.AgentDiscoveryService
}

func NewAgentDiscoveryHandler(svc *service.AgentDiscoveryService) *AgentDiscoveryHandler {
	return &AgentDiscoveryHandler{svc: svc}
}

// Report is called by log-agent (public; uses agent token inside payload).
func (h *AgentDiscoveryHandler) Report(c *gin.Context) {
	handleJSON(c, h.svc.Report)
}

// List is used by UI (authz) to fetch discovery items for a project/server.
func (h *AgentDiscoveryHandler) List(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q service.AgentDiscoveryListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID
	list, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

