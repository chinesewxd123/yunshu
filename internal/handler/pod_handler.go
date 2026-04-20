package handler

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
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

// NewPodHandler 创建相关逻辑。
func NewPodHandler(svc *service.K8sPodService) *PodHandler {
	return &PodHandler{svc: svc}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *PodHandler) List(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.PodListQuery) (gin.H, error) {
		list, err := h.svc.List(ctx, query)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// Logs 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) Logs(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.PodLogsQuery) (gin.H, error) {
		logs, err := h.svc.GetLogs(ctx, query)
		if err != nil {
			return nil, err
		}
		return gin.H{"logs": logs}, nil
	})
}

// LogsDownload 处理对应的 HTTP 请求并返回统一响应。
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

// LogsStream 处理对应的 HTTP 请求并返回统一响应。
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

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *PodHandler) Delete(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "deleted"}, h.svc.Delete)
}

// Detail 查询详情对应的 HTTP 接口处理逻辑。
func (h *PodHandler) Detail(c *gin.Context) {
	handleQuery(c, h.svc.Detail)
}

// Events 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) Events(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.PodEventQuery) (gin.H, error) {
		list, err := h.svc.Events(ctx, query)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list, "ts": time.Now().Unix()}, nil
	})
}

// Exec 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) Exec(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.PodExecRequest) (gin.H, error) {
		out, err := h.svc.Exec(ctx, req)
		if err != nil {
			return nil, err
		}
		return gin.H{"output": out}, nil
	})
}

// Restart 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) Restart(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "restarted"}, h.svc.Restart)
}

// CreateByYAML 创建对应的 HTTP 接口处理逻辑。
func (h *PodHandler) CreateByYAML(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "created"}, h.svc.CreateByYAML)
}

// CreateSimple 创建对应的 HTTP 接口处理逻辑。
func (h *PodHandler) CreateSimple(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "created"}, h.svc.CreateSimple)
}

// UpdateSimple 更新对应的 HTTP 接口处理逻辑。
func (h *PodHandler) UpdateSimple(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "updated"}, h.svc.UpdateSimple)
}

// ListFiles 查询列表对应的 HTTP 接口处理逻辑。
func (h *PodHandler) ListFiles(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.PodFileQuery) (gin.H, error) {
		list, err := h.svc.ListFiles(ctx, query)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// ReadFile 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) ReadFile(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, query service.PodFileQuery) (gin.H, error) {
		data, err := h.svc.ReadFile(ctx, query)
		if err != nil {
			return nil, err
		}
		return gin.H{"content": string(data)}, nil
	})
}

// DownloadFile 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) DownloadFile(c *gin.Context) {
	var query service.PodFileQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.svc.ReadFile(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := filepath.Base(strings.TrimSpace(query.Path))
	if filename == "" || filename == "." || filename == "/" {
		filename = "download.bin"
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	c.Data(200, "application/octet-stream", data)
}

// DeleteFile 删除对应的 HTTP 接口处理逻辑。
func (h *PodHandler) DeleteFile(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "deleted"}, h.svc.DeleteFile)
}

// UploadFile 处理对应的 HTTP 请求并返回统一响应。
func (h *PodHandler) UploadFile(c *gin.Context) {
	clusterID, err := strconv.ParseUint(strings.TrimSpace(c.PostForm("cluster_id")), 10, 64)
	if err != nil || clusterID == 0 {
		response.Error(c, apperror.BadRequest("集群 ID 不合法"))
		return
	}
	query := service.PodFileQuery{
		ClusterID: uint(clusterID),
		Namespace: strings.TrimSpace(c.PostForm("namespace")),
		Name:      strings.TrimSpace(c.PostForm("name")),
		Container: strings.TrimSpace(c.PostForm("container")),
		Path:      strings.TrimSpace(c.PostForm("path")),
	}
	if query.Namespace == "" || query.Name == "" {
		response.Error(c, apperror.BadRequest("命名空间和名称不能为空"))
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("请上传文件"))
		return
	}
	file, err := fh.Open()
	if err != nil {
		response.Error(c, apperror.BadRequest("读取上传文件失败"))
		return
	}
	defer file.Close()
	if err := h.svc.UploadFile(c.Request.Context(), query, fh.Filename, file); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "uploaded"})
}
