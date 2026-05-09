package handler

import (
	"strconv"
	"strings"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sDiscoveryHandler struct {
	svc *service.K8sDiscoveryService
}

func NewK8sDiscoveryHandler(svc *service.K8sDiscoveryService) *K8sDiscoveryHandler {
	return &K8sDiscoveryHandler{svc: svc}
}

// ListAPIResources GET /clusters/:id/api-resources ?namespaced=true|false
func (h *K8sDiscoveryHandler) ListAPIResources(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var ns *bool
	switch strings.TrimSpace(strings.ToLower(c.Query("namespaced"))) {
	case "true":
		t := true
		ns = &t
	case "false":
		f := false
		ns = &f
	}
	list, err := h.svc.ListAPIResources(c.Request.Context(), id, ns)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list, "cluster_id": strconv.FormatUint(uint64(id), 10)})
}
