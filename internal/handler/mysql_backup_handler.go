package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type MysqlBackupHandler struct {
	svc *service.MysqlBackupService
}

func NewMysqlBackupHandler(svc *service.MysqlBackupService) *MysqlBackupHandler {
	return &MysqlBackupHandler{svc: svc}
}

func (h *MysqlBackupHandler) ListMysqldumpOptions(c *gin.Context) {
	response.Success(c, h.svc.ListMysqldumpOptions())
}

func (h *MysqlBackupHandler) ListInstances(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.MysqlBackupInstanceListQuery) (*pagination.Result[service.MysqlBackupInstanceItem], error) {
		q.ProjectID = projectID
		return h.svc.ListInstances(ctx, q)
	})
}

func (h *MysqlBackupHandler) CreateInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req service.MysqlBackupInstanceUpsertRequest) (*service.MysqlBackupInstanceItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertInstance(ctx, 0, req)
	})
}

func (h *MysqlBackupHandler) UpdateInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.MysqlBackupInstanceUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ProjectID = projectID
	item, err := h.svc.UpsertInstance(c.Request.Context(), instanceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *MysqlBackupHandler) DeleteInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteInstance(c.Request.Context(), projectID, instanceID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *MysqlBackupHandler) PingInstance(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ok, msg, err := h.svc.PingInstance(c.Request.Context(), projectID, instanceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": ok, "message": msg})
}

func (h *MysqlBackupHandler) CheckRemote(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.CheckRemoteBackup(c.Request.Context(), projectID, instanceID, -1)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *MysqlBackupHandler) RunBackup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	instanceID, err := parseUintParam(c, "instanceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.RunBackup(c.Request.Context(), projectID, instanceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *MysqlBackupHandler) ListJobs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.MysqlBackupJobListQuery) (*pagination.Result[model.MysqlBackupJob], error) {
		q.ProjectID = projectID
		return h.svc.ListJobs(ctx, q)
	})
}

func (h *MysqlBackupHandler) PresignJob(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	jobID, err := parseUintParam(c, "jobId")
	if err != nil {
		response.Error(c, err)
		return
	}
	url, err := h.svc.PresignDownload(c.Request.Context(), projectID, jobID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"url": url})
}
