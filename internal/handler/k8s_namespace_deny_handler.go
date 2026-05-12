package handler

import (
	"strconv"
	"strings"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sNamespaceDenyHandler struct {
	svc *service.K8sNamespaceDenyService
}

func NewK8sNamespaceDenyHandler(svc *service.K8sNamespaceDenyService) *K8sNamespaceDenyHandler {
	return &K8sNamespaceDenyHandler{svc: svc}
}

func (h *K8sNamespaceDenyHandler) List(c *gin.Context) {
	kind := strings.TrimSpace(c.Query("principal_kind"))
	ref := strings.TrimSpace(c.Query("principal_ref"))
	if kind == "" && ref == "" {
		if legacy := strings.TrimSpace(c.Query("role_code")); legacy != "" {
			kind = "role"
			ref = legacy
		}
	}
	var clusterID uint
	if raw := strings.TrimSpace(c.Query("cluster_id")); raw != "" {
		if n, err := strconv.ParseUint(raw, 10, 32); err == nil {
			clusterID = uint(n)
		}
	}
	list, err := h.svc.List(c.Request.Context(), kind, ref, clusterID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *K8sNamespaceDenyHandler) Create(c *gin.Context) {
	ServeJSON(c, h.svc.Create)
}

func (h *K8sNamespaceDenyHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
