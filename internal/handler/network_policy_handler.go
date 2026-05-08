package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type NetworkPolicyHandler struct {
	svc *service.K8sNetworkPolicyService
}

func NewNetworkPolicyHandler(svc *service.K8sNetworkPolicyService) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{svc: svc}
}

func (h *NetworkPolicyHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

func (h *NetworkPolicyHandler) Detail(c *gin.Context) {
	ServeQuery(c, h.svc.Detail)
}

func (h *NetworkPolicyHandler) Apply(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.Apply)
}

func (h *NetworkPolicyHandler) Delete(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.Delete)
}
