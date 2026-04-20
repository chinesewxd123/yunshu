package handler

import (
	"context"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type DictEntryHandler struct {
	svc *service.DictEntryService
}

func NewDictEntryHandler(svc *service.DictEntryService) *DictEntryHandler {
	return &DictEntryHandler{svc: svc}
}

func (h *DictEntryHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.List)
}

func (h *DictEntryHandler) Create(c *gin.Context) {
	handleJSONCreated(c, h.svc.Create)
}

func (h *DictEntryHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.DictEntryUpdateRequest) (*model.DictEntry, error) {
		item, err := h.svc.Update(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return item, nil
	})
}

func (h *DictEntryHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *DictEntryHandler) Options(c *gin.Context) {
	dictType := c.Param("dictType")
	list, err := h.svc.Options(c.Request.Context(), dictType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}
