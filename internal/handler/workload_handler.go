package handler

import (
	"context"

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
	handleQuery(c, h.svc.ListDeployments)
}

func (h *WorkloadHandler) DeploymentDetail(c *gin.Context) {
	handleQuery(c, h.svc.DeploymentDetail)
}

func (h *WorkloadHandler) DeploymentScale(c *gin.Context) {
	handleJSONOK(c, true, h.svc.DeploymentScale)
}

func (h *WorkloadHandler) DeploymentRestart(c *gin.Context) {
	handleJSONOK(c, true, h.svc.DeploymentRestart)
}

func (h *WorkloadHandler) DeleteDeployment(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteDeployment)
}

// StatefulSets
func (h *WorkloadHandler) ListStatefulSets(c *gin.Context) {
	handleQuery(c, h.svc.ListStatefulSets)
}

func (h *WorkloadHandler) StatefulSetDetail(c *gin.Context) {
	handleQuery(c, h.svc.StatefulSetDetail)
}

func (h *WorkloadHandler) StatefulSetScale(c *gin.Context) {
	handleJSONOK(c, true, h.svc.StatefulSetScale)
}

func (h *WorkloadHandler) StatefulSetRestart(c *gin.Context) {
	handleJSONOK(c, true, h.svc.StatefulSetRestart)
}

func (h *WorkloadHandler) DeleteStatefulSet(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteStatefulSet)
}

// DaemonSets
func (h *WorkloadHandler) ListDaemonSets(c *gin.Context) {
	handleQuery(c, h.svc.ListDaemonSets)
}

func (h *WorkloadHandler) DaemonSetDetail(c *gin.Context) {
	handleQuery(c, h.svc.DaemonSetDetail)
}

func (h *WorkloadHandler) DaemonSetRestart(c *gin.Context) {
	handleJSONOK(c, true, h.svc.DaemonSetRestart)
}

func (h *WorkloadHandler) DeleteDaemonSet(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteDaemonSet)
}

// Jobs
func (h *WorkloadHandler) ListJobs(c *gin.Context) {
	handleQuery(c, h.svc.ListJobs)
}

func (h *WorkloadHandler) JobDetail(c *gin.Context) {
	handleQuery(c, h.svc.JobDetail)
}

func (h *WorkloadHandler) DeleteJob(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteJob)
}

// CronJobs
func (h *WorkloadHandler) ListCronJobs(c *gin.Context) {
	handleQuery(c, h.svc.ListCronJobs)
}

func (h *WorkloadHandler) ListCronJobsV2(c *gin.Context) {
	handleQuery(c, h.svc.ListCronJobsV2)
}

func (h *WorkloadHandler) CronJobDetail(c *gin.Context) {
	handleQuery(c, h.svc.CronJobDetail)
}

func (h *WorkloadHandler) CronJobSuspend(c *gin.Context) {
	handleJSONOK(c, true, h.svc.CronJobSuspend)
}

func (h *WorkloadHandler) CronJobTrigger(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.CronJobTriggerRequest) (gin.H, error) {
		name, err := h.svc.CronJobTrigger(ctx, req)
		return gin.H{"job_name": name}, err
	})
}

func (h *WorkloadHandler) DeleteCronJob(c *gin.Context) {
	handleQueryOK(c, true, h.svc.DeleteCronJob)
}

func (h *WorkloadHandler) JobRerun(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.JobRerunRequest) (gin.H, error) {
		name, err := h.svc.JobRerun(ctx, req)
		return gin.H{"job_name": name}, err
	})
}

// related pods
func (h *WorkloadHandler) DeploymentPods(c *gin.Context) {
	handleQuery(c, h.svc.DeploymentPods)
}

func (h *WorkloadHandler) StatefulSetPods(c *gin.Context) {
	handleQuery(c, h.svc.StatefulSetPods)
}

func (h *WorkloadHandler) DaemonSetPods(c *gin.Context) {
	handleQuery(c, h.svc.DaemonSetPods)
}

func (h *WorkloadHandler) JobPods(c *gin.Context) {
	handleQuery(c, h.svc.JobPods)
}

func (h *WorkloadHandler) CronJobPods(c *gin.Context) {
	handleQuery(c, h.svc.CronJobPods)
}

// shared
func (h *WorkloadHandler) Apply(c *gin.Context) {
	handleJSONOK(c, true, h.svc.Apply)
}
