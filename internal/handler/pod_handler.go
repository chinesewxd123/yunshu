package handler

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type PodHandler struct {
	svc *service.K8sPodService
}

func NewPodHandler(svc *service.K8sPodService) *PodHandler {
	return &PodHandler{svc: svc}
}

func (h *PodHandler) List(c *gin.Context) {
	var query service.PodListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	list, err := h.svc.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *PodHandler) Logs(c *gin.Context) {
	var query service.PodLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	logs, err := h.svc.GetLogs(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"logs": logs})
}

func (h *PodHandler) LogsDownload(c *gin.Context) {
	var query service.PodLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	logs, err := h.svc.GetLogs(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := fmt.Sprintf("%s-%s.log", query.Namespace, query.Name)
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	c.String(200, logs)
}

func (h *PodHandler) LogsStream(c *gin.Context) {
	var query service.PodLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)
	c.Writer.Flush()

	_ = h.svc.StreamLogs(c.Request.Context(), query, func(line string) error {
		if _, err := c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", strings.TrimRight(line, "\r\n"))); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	})
}

func (h *PodHandler) Delete(c *gin.Context) {
	var req service.PodDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *PodHandler) Detail(c *gin.Context) {
	var query service.PodDetailQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PodHandler) Events(c *gin.Context) {
	var query service.PodEventQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	list, err := h.svc.Events(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list, "ts": time.Now().Unix()})
}

func (h *PodHandler) Exec(c *gin.Context) {
	var req service.PodExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	out, err := h.svc.Exec(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"output": out})
}

func (h *PodHandler) Restart(c *gin.Context) {
	var req service.PodRestartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.Restart(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "restarted"})
}

func (h *PodHandler) CreateByYAML(c *gin.Context) {
	var req service.PodCreateYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.CreateByYAML(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "created"})
}

func (h *PodHandler) CreateSimple(c *gin.Context) {
	var req service.PodCreateSimpleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.CreateSimple(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "created"})
}

func (h *PodHandler) UpdateSimple(c *gin.Context) {
	var req service.PodCreateSimpleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.UpdateSimple(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "updated"})
}
