package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type StorageHandler struct {
	svc *service.K8sStorageService
}

func NewStorageHandler(svc *service.K8sStorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

func (h *StorageHandler) ListPVs(c *gin.Context) {
	handleQuery(c, h.svc.ListPVs)
}

func (h *StorageHandler) ListPVCs(c *gin.Context) {
	handleQuery(c, h.svc.ListPVCs)
}

func (h *StorageHandler) ListStorageClasses(c *gin.Context) {
	handleQuery(c, h.svc.ListStorageClasses)
}

func (h *StorageHandler) Detail(c *gin.Context) {
	handleQueryWithKind(c, h.svc.Detail)
}

func (h *StorageHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

func (h *StorageHandler) Delete(c *gin.Context) {
	handleQueryWithKindOK(c, true, h.svc.Delete)
}
