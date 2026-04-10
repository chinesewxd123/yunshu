package handler

import (
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type WorkloadHandler struct {
	svc *service.K8sWorkloadService
}

func NewWorkloadHandler(svc *service.K8sWorkloadService) *WorkloadHandler {
	return &WorkloadHandler{svc: svc}
}

// Deployments
func (h *WorkloadHandler) ListDeployments(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListDeployments(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) DeploymentDetail(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.DeploymentDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkloadHandler) DeploymentScale(c *gin.Context) {
	var req service.WorkloadScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeploymentScale(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) DeploymentRestart(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindJSON(&q); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeploymentRestart(c.Request.Context(), q); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) DeleteDeployment(c *gin.Context) {
	var req service.NamespacedDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteDeployment(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// StatefulSets
func (h *WorkloadHandler) ListStatefulSets(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListStatefulSets(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) StatefulSetDetail(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.StatefulSetDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkloadHandler) StatefulSetScale(c *gin.Context) {
	var req service.WorkloadScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.StatefulSetScale(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) StatefulSetRestart(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindJSON(&q); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.StatefulSetRestart(c.Request.Context(), q); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) DeleteStatefulSet(c *gin.Context) {
	var req service.NamespacedDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteStatefulSet(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// DaemonSets
func (h *WorkloadHandler) ListDaemonSets(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListDaemonSets(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) DaemonSetDetail(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.DaemonSetDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkloadHandler) DaemonSetRestart(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindJSON(&q); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DaemonSetRestart(c.Request.Context(), q); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) DeleteDaemonSet(c *gin.Context) {
	var req service.NamespacedDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteDaemonSet(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// Jobs
func (h *WorkloadHandler) ListJobs(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListJobs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) JobDetail(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.JobDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkloadHandler) DeleteJob(c *gin.Context) {
	var req service.NamespacedDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteJob(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

// CronJobs
func (h *WorkloadHandler) ListCronJobs(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListCronJobs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) ListCronJobsV2(c *gin.Context) {
	var q service.NamespacedListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListCronJobsV2(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) CronJobDetail(c *gin.Context) {
	var q service.NamespacedDetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.CronJobDetail(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkloadHandler) CronJobSuspend(c *gin.Context) {
	var req service.CronJobSuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.CronJobSuspend(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) CronJobTrigger(c *gin.Context) {
	var req service.CronJobTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	name, err := h.svc.CronJobTrigger(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"job_name": name})
}

func (h *WorkloadHandler) DeleteCronJob(c *gin.Context) {
	var req service.NamespacedDeleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteCronJob(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}

func (h *WorkloadHandler) JobRerun(c *gin.Context) {
	var req service.JobRerunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	name, err := h.svc.JobRerun(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"job_name": name})
}

// related pods
func (h *WorkloadHandler) DeploymentPods(c *gin.Context) {
	var q service.RelatedPodsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.DeploymentPods(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) StatefulSetPods(c *gin.Context) {
	var q service.RelatedPodsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.StatefulSetPods(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) DaemonSetPods(c *gin.Context) {
	var q service.RelatedPodsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.DaemonSetPods(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) JobPods(c *gin.Context) {
	var q service.RelatedPodsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.JobPods(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *WorkloadHandler) CronJobPods(c *gin.Context) {
	var q service.RelatedPodsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.CronJobPods(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// shared
func (h *WorkloadHandler) Apply(c *gin.Context) {
	var req service.NamespacedApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Apply(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, true)
}
