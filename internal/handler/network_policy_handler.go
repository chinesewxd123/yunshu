package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type NetworkPolicyHandler struct {
	svc *service.K8sNetworkPolicyService
}

func NewNetworkPolicyHandler(svc *service.K8sNetworkPolicyService) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{svc: svc}
}

func (h *NetworkPolicyHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *NetworkPolicyHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

func (h *NetworkPolicyHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *NetworkPolicyHandler) Delete(c *gin.Context) {
	handleQueryOK(c, true, h.svc.Delete)
}
