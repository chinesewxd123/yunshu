package handler

import (
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type DictEntryHandler struct {
	svc *service.DictEntryService
}

func NewDictEntryHandler(svc *service.DictEntryService) *DictEntryHandler {
	return &DictEntryHandler{svc: svc}
}

func (h *DictEntryHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.List)
}

func (h *DictEntryHandler) Create(c *gin.Context) {
	ServeJSON201(c, h.svc.Create)
}

func (h *DictEntryHandler) Update(c *gin.Context) {
	ServePatch(c, h.svc.Update, "")
}

func (h *DictEntryHandler) Delete(c *gin.Context) {
	ServeDelete(c, h.svc.Delete, "")
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
