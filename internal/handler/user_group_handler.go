package handler

import (
	"context"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type UserGroupHandler struct {
	svc *service.UserGroupService
}

func NewUserGroupHandler(svc *service.UserGroupService) *UserGroupHandler {
	return &UserGroupHandler{svc: svc}
}

func (h *UserGroupHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

func (h *UserGroupHandler) Detail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *UserGroupHandler) Create(c *gin.Context) {
	ServeJSON201(c, h.svc.Create)
}

func (h *UserGroupHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.UserGroupUpdateRequest) (*service.UserGroupItem, error) {
		return h.svc.Update(ctx, id, req)
	})
}

func (h *UserGroupHandler) Delete(c *gin.Context) {
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

func (h *UserGroupHandler) AssignUsers(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"message": "ok"}, func(ctx context.Context, req service.UserGroupAssignUsersRequest) error {
		return h.svc.AssignUsers(ctx, id, req)
	})
}
