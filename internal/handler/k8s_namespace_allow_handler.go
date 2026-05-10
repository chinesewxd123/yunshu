package handler

import (
	"strconv"
	"strings"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sNamespaceAllowHandler struct {
	svc *service.K8sNamespaceAllowService
}

func NewK8sNamespaceAllowHandler(svc *service.K8sNamespaceAllowService) *K8sNamespaceAllowHandler {
	return &K8sNamespaceAllowHandler{svc: svc}
}

func (h *K8sNamespaceAllowHandler) List(c *gin.Context) {
	kind := strings.TrimSpace(c.Query("principal_kind"))
	ref := strings.TrimSpace(c.Query("principal_ref"))
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

func (h *K8sNamespaceAllowHandler) Create(c *gin.Context) {
	ServeJSON(c, h.svc.Create)
}

func (h *K8sNamespaceAllowHandler) Delete(c *gin.Context) {
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
