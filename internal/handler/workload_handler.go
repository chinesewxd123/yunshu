package handler

import (
	"context"

	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type WorkloadHandler struct {
	svc *service.K8sWorkloadService
}

// NewWorkloadHandler 创建相关逻辑。
func NewWorkloadHandler(svc *service.K8sWorkloadService) *WorkloadHandler {
	return &WorkloadHandler{svc: svc}
}

// Deployments
func (h *WorkloadHandler) ListDeployments(c *gin.Context) {
	ServeQuery(c, h.svc.ListDeployments)
}

// DeploymentDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DeploymentDetail(c *gin.Context) {
	ServeQuery(c, h.svc.DeploymentDetail)
}

// DeploymentScale 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DeploymentScale(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.DeploymentScale)
}

// DeploymentPatchContainerResources 垂直扩缩（容器 requests/limits）。
func (h *WorkloadHandler) DeploymentPatchContainerResources(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.DeploymentPatchContainerResources)
}

// DeploymentRestart 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DeploymentRestart(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.DeploymentRestart)
}

// DeleteDeployment 删除对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) DeleteDeployment(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.DeleteDeployment)
}

// StatefulSets
func (h *WorkloadHandler) ListStatefulSets(c *gin.Context) {
	ServeQuery(c, h.svc.ListStatefulSets)
}

// StatefulSetDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) StatefulSetDetail(c *gin.Context) {
	ServeQuery(c, h.svc.StatefulSetDetail)
}

// StatefulSetScale 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) StatefulSetScale(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.StatefulSetScale)
}

// StatefulSetPatchContainerResources 垂直扩缩（容器 requests/limits）。
func (h *WorkloadHandler) StatefulSetPatchContainerResources(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.StatefulSetPatchContainerResources)
}

// StatefulSetRestart 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) StatefulSetRestart(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.StatefulSetRestart)
}

// DeleteStatefulSet 删除对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) DeleteStatefulSet(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.DeleteStatefulSet)
}

// DaemonSets
func (h *WorkloadHandler) ListDaemonSets(c *gin.Context) {
	ServeQuery(c, h.svc.ListDaemonSets)
}

// DaemonSetDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DaemonSetDetail(c *gin.Context) {
	ServeQuery(c, h.svc.DaemonSetDetail)
}

// DaemonSetRestart 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DaemonSetRestart(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.DaemonSetRestart)
}

// DaemonSetPatchContainerResources 垂直扩缩（容器 requests/limits）。
func (h *WorkloadHandler) DaemonSetPatchContainerResources(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.DaemonSetPatchContainerResources)
}

// DeleteDaemonSet 删除对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) DeleteDaemonSet(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.DeleteDaemonSet)
}

// Jobs
func (h *WorkloadHandler) ListJobs(c *gin.Context) {
	ServeQuery(c, h.svc.ListJobs)
}

// JobDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) JobDetail(c *gin.Context) {
	ServeQuery(c, h.svc.JobDetail)
}

// JobPatchContainerResources 垂直扩缩（容器 requests/limits）。
func (h *WorkloadHandler) JobPatchContainerResources(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.JobPatchContainerResources)
}

// DeleteJob 删除对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) DeleteJob(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.DeleteJob)
}

// CronJobs
func (h *WorkloadHandler) ListCronJobs(c *gin.Context) {
	ServeQuery(c, h.svc.ListCronJobs)
}

// ListCronJobsV2 查询列表对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) ListCronJobsV2(c *gin.Context) {
	ServeQuery(c, h.svc.ListCronJobsV2)
}

// CronJobDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) CronJobDetail(c *gin.Context) {
	ServeQuery(c, h.svc.CronJobDetail)
}

// CronJobPatchContainerResources 垂直扩缩（容器 requests/limits）。
func (h *WorkloadHandler) CronJobPatchContainerResources(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.CronJobPatchContainerResources)
}

// CronJobSuspend 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) CronJobSuspend(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.CronJobSuspend)
}

// CronJobTrigger 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) CronJobTrigger(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.CronJobTriggerRequest) (gin.H, error) {
		name, err := h.svc.CronJobTrigger(ctx, req)
		return gin.H{"job_name": name}, err
	})
}

// DeleteCronJob 删除对应的 HTTP 接口处理逻辑。
func (h *WorkloadHandler) DeleteCronJob(c *gin.Context) {
	ServeQueryOK(c, true, h.svc.DeleteCronJob)
}

// JobRerun 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) JobRerun(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.JobRerunRequest) (gin.H, error) {
		name, err := h.svc.JobRerun(ctx, req)
		return gin.H{"job_name": name}, err
	})
}

// related pods
func (h *WorkloadHandler) DeploymentPods(c *gin.Context) {
	ServeQuery(c, h.svc.DeploymentPods)
}

// StatefulSetPods 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) StatefulSetPods(c *gin.Context) {
	ServeQuery(c, h.svc.StatefulSetPods)
}

// DaemonSetPods 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) DaemonSetPods(c *gin.Context) {
	ServeQuery(c, h.svc.DaemonSetPods)
}

// JobPods 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) JobPods(c *gin.Context) {
	ServeQuery(c, h.svc.JobPods)
}

// CronJobPods 处理对应的 HTTP 请求并返回统一响应。
func (h *WorkloadHandler) CronJobPods(c *gin.Context) {
	ServeQuery(c, h.svc.CronJobPods)
}

// shared
func (h *WorkloadHandler) Apply(c *gin.Context) {
	ServeJSONOK(c, true, h.svc.Apply)
}
