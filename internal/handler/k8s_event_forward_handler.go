package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sEventForwardHandler struct {
	svc *service.K8sEventForwardAdminService
}

func NewK8sEventForwardHandler(svc *service.K8sEventForwardAdminService) *K8sEventForwardHandler {
	return &K8sEventForwardHandler{svc: svc}
}

func (h *K8sEventForwardHandler) ListRules(c *gin.Context) {
	ServeQuery(c, h.svc.ListRules)
}

func (h *K8sEventForwardHandler) GetRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	rule, err := h.svc.GetRule(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rule)
}

func (h *K8sEventForwardHandler) CreateRule(c *gin.Context) {
	ServeJSON201(c, func(ctx context.Context, req model.K8sEventForwardRule) (*model.K8sEventForwardRule, error) {
		if err := h.svc.CreateRule(ctx, &req); err != nil {
			return nil, err
		}
		return &req, nil
	})
}

func (h *K8sEventForwardHandler) UpdateRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req model.K8sEventForwardRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ID = id
	if err := h.svc.UpdateRule(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, req)
}

func (h *K8sEventForwardHandler) DeleteRule(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteRule(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *K8sEventForwardHandler) GetSettings(c *gin.Context) {
	st, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, st)
}

func (h *K8sEventForwardHandler) UpdateSettings(c *gin.Context) {
	var req model.K8sEventForwardSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.UpdateSettings(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, req)
}
