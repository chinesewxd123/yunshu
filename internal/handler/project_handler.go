package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	svc *service.ProjectMgmtService
}

func NewProjectHandler(svc *service.ProjectMgmtService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.ListProjects)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	handleJSON(c, h.svc.CreateProject)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.ProjectUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	data, err := h.svc.UpdateProject(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteProject(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectHandler) ListServers(c *gin.Context) {
	handleQuery(c, h.svc.ListServers)
}

func (h *ProjectHandler) UpsertServer(c *gin.Context) {
	handleJSON(c, h.svc.UpsertServer)
}

func (h *ProjectHandler) DeleteServer(c *gin.Context) {
	id, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteServer(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectHandler) TestServer(c *gin.Context) {
	handleJSON(c, h.svc.TestServerConnectivity)
}

func (h *ProjectHandler) ImportServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("file upload failed"))
		return
	}
	defer file.Close()
	n, err := h.svc.ImportServersFromExcel(c.Request.Context(), projectID, file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"imported": n})
}

func (h *ProjectHandler) ExportServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	keyword := c.Query("keyword")
	f, err := h.svc.ExportServersToExcel(c.Request.Context(), projectID, keyword)
	if err != nil {
		response.Error(c, err)
		return
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := fmt.Sprintf("project-%d-servers.xlsx", projectID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *ProjectHandler) ListServices(c *gin.Context) {
	handleQuery(c, h.svc.ListServices)
}

func (h *ProjectHandler) UpsertService(c *gin.Context) {
	handleJSON(c, h.svc.UpsertService)
}

func (h *ProjectHandler) DeleteService(c *gin.Context) {
	id, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectHandler) ListLogSources(c *gin.Context) {
	handleQuery(c, h.svc.ListLogSources)
}

func (h *ProjectHandler) UpsertLogSource(c *gin.Context) {
	handleJSON(c, h.svc.UpsertLogSource)
}

func (h *ProjectHandler) DeleteLogSource(c *gin.Context) {
	id, err := parseUintParam(c, "logSourceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteLogSource(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectHandler) StreamLogs(c *gin.Context) {
	// SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var q service.LogStreamQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID

	streamKey := service.BuildLogStreamKey(q.ProjectID, q.ServerID, q.LogSourceID)
	var includeRe *regexp.Regexp
	if q.Include != nil && strings.TrimSpace(*q.Include) != "" {
		re, err := regexp.Compile(*q.Include)
		if err != nil {
			response.Error(c, apperror.BadRequest("invalid include regex"))
			return
		}
		includeRe = re
	}
	var excludeRe *regexp.Regexp
	if q.Exclude != nil && strings.TrimSpace(*q.Exclude) != "" {
		re, err := regexp.Compile(*q.Exclude)
		if err != nil {
			response.Error(c, apperror.BadRequest("invalid exclude regex"))
			return
		}
		excludeRe = re
	}
	hl := ""
	if q.Highlight != nil {
		hl = strings.TrimSpace(*q.Highlight)
	}
	targetFilePath := ""
	if q.FilePath != nil {
		targetFilePath = strings.TrimSpace(*q.FilePath)
	}
	replayLines := q.TailLines
	if replayLines <= 0 {
		replayLines = 200
	}

	ch, cancelSub := service.AgentLogBroker.Subscribe(streamKey, replayLines)
	defer cancelSub()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			_, _ = c.Writer.WriteString("event: ping\ndata: {}\n\n")
			c.Writer.Flush()
		case event := <-ch:
			if targetFilePath != "" && strings.TrimSpace(event.FilePath) != targetFilePath {
				continue
			}
			line := event.Line
			if includeRe != nil && !includeRe.MatchString(line) {
				continue
			}
			if excludeRe != nil && excludeRe.MatchString(line) {
				continue
			}
			if hl != "" && strings.Contains(line, hl) {
				line = strings.ReplaceAll(line, hl, "\x1b[31m"+hl+"\x1b[0m")
			}
			if len(line) > 4096 {
				line = line[:4096] + " ...<truncated>"
			}
			c.SSEvent("log", gin.H{"line": line, "file_path": strings.TrimSpace(event.FilePath)})
			c.Writer.Flush()
		}
	}
}

func (h *ProjectHandler) ExportLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q service.LogExportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID
	data, filename, err := h.svc.ExportLogs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

func (h *ProjectHandler) ListLogFiles(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q service.RemoteLogFileQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID
	files, err := h.svc.ListRemoteLogFiles(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": files})
}

func (h *ProjectHandler) ListLogUnits(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q service.RemoteLogUnitQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID
	list, err := h.svc.ListRemoteLogUnits(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}
