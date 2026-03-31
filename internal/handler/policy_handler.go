package handler

import (
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type PolicyHandler struct {
	service *service.PolicyService
}

func NewPolicyHandler(service *service.PolicyService) *PolicyHandler {
	return &PolicyHandler{service: service}
}

// List godoc
// @Summary List policies
// @Description List current role-permission policy bindings.
// @Tags Policy
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=[]service.PolicyItemResponse} "success"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/policies [get]
func (h *PolicyHandler) List(c *gin.Context) {
	data, err := h.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// Grant godoc
// @Summary Grant policy
// @Description Bind one permission to one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Grant policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/policies [post]
func (h *PolicyHandler) Grant(c *gin.Context) {
	var req service.PolicyGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.service.Grant(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "granted"})
}

// Revoke godoc
// @Summary Revoke policy
// @Description Remove one permission from one role.
// @Tags Policy
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.PolicyGrantRequest true "Revoke policy request"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "unauthorized"
// @Failure 403 {object} response.Body "forbidden"
// @Failure 404 {object} response.Body "resource not found"
// @Failure 500 {object} response.Body "internal server error"
// @Router /api/v1/policies [delete]
func (h *PolicyHandler) Revoke(c *gin.Context) {
	var req service.PolicyGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.service.Revoke(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "revoked"})
}
