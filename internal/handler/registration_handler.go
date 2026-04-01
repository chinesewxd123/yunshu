package handler

import (
	"strconv"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type RegistrationHandler struct {
	service *service.RegistrationService
}

func NewRegistrationHandler(svc *service.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{service: svc}
}

func (h *RegistrationHandler) Apply(c *gin.Context) {
	var req service.ApplyRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.service.Apply(c.Request.Context(), req); err != nil {
		response.Error(c, apperror.Conflict(err.Error()))
		return
	}
	response.Success(c, nil)
}

func (h *RegistrationHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	var status *int
	if s := c.Query("status"); s != "" {
		v, err := strconv.Atoi(s)
		if err == nil {
			status = &v
		}
	}

	list, total, err := h.service.List(c.Request.Context(), keyword, status, page, pageSize)
	if err != nil {
		response.Error(c, apperror.Internal(err.Error()))
		return
	}
	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *RegistrationHandler) Review(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, apperror.BadRequest("invalid id"))
		return
	}

	var req service.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("unauthorized"))
		return
	}

	if err := h.service.Review(c.Request.Context(), uint(id), user.ID, req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	statusText := "approved"
	if req.Status == 2 {
		statusText = "rejected"
	}
	response.Success(c, gin.H{"message": "registration request has been " + statusText})
}
