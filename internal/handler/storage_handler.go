package handler

import (
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type StorageHandler struct {
	svc *service.K8sStorageService
}

// NewStorageHandler 创建相关逻辑。
func NewStorageHandler(svc *service.K8sStorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

// ListPVs 查询列表对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) ListPVs(c *gin.Context) {
	handleQuery(c, h.svc.ListPVs)
}

// ListPVCs 查询列表对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) ListPVCs(c *gin.Context) {
	handleQuery(c, h.svc.ListPVCs)
}

// ListStorageClasses 查询列表对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) ListStorageClasses(c *gin.Context) {
	handleQuery(c, h.svc.ListStorageClasses)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) Detail(c *gin.Context) {
	handleQueryWithKind(c, h.svc.Detail)
}

// Apply 提交申请对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *StorageHandler) Delete(c *gin.Context) {
	handleQueryWithKindOK(c, true, h.svc.Delete)
}
