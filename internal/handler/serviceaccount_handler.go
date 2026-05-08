package handler

import (
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ServiceAccountHandler struct {
	svc *service.K8sServiceAccountService
}

func NewServiceAccountHandler(svc *service.K8sServiceAccountService) *ServiceAccountHandler {
	return &ServiceAccountHandler{svc: svc}
}

func (h *ServiceAccountHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

func (h *ServiceAccountHandler) Detail(c *gin.Context) {
	ServeQuery(c, h.svc.Detail)
}

func (h *ServiceAccountHandler) Apply(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.Apply)
}

func (h *ServiceAccountHandler) Delete(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.Delete)
}
