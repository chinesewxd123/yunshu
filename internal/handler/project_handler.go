package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	svc             *service.ProjectMgmtService
	projectClient   pb.ProjectServerServiceClient
	logSourceClient pb.LogSourceServiceClient
}

// NewProjectHandler 创建相关逻辑。
func NewProjectHandler(svc *service.ProjectMgmtService, projectClient pb.ProjectServerServiceClient, logSourceClient pb.LogSourceServiceClient) *ProjectHandler {
	return &ProjectHandler{svc: svc, projectClient: projectClient, logSourceClient: logSourceClient}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) List(c *gin.Context) {
	handleQuery(c, h.svc.ListProjects)
}

// Create 创建对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) Create(c *gin.Context) {
	u, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("未登录"))
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ProjectCreateRequest) (*service.ProjectItem, error) {
		return h.svc.CreateProject(ctx, u.ID, req)
	})
}

// Update 更新对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ProjectUpdateRequest) (*service.ProjectItem, error) {
		return h.svc.UpdateProject(ctx, id, req)
	})
}

// Delete 删除对应的 HTTP 接口处理逻辑。
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

// ListServers 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleQuery(c, func(ctx context.Context, q service.ServerListQuery) (*pagination.Result[service.ServerItem], error) {
		q.ProjectID = projectID
		return h.svc.ListServers(ctx, q)
	})
}

// UpsertServer 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertServer(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ServerUpsertRequest) (*service.ServerItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertServer(ctx, req)
	})
}

// DeleteServer 删除对应的 HTTP 接口处理逻辑。
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

// ServerDetail 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) ServerDetail(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.GetServer(c.Request.Context(), serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if data.ProjectID != projectID {
		response.Error(c, apperror.BadRequest("服务器不属于当前项目"))
		return
	}
	response.Success(c, data)
}

// ExecServerCommand 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) ExecServerCommand(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ServerExecRequest) (*service.ServerExecResult, error) {
		req.ProjectID = projectID
		req.ServerID = serverID
		return h.svc.ExecServerCommand(ctx, req)
	})
}

// ListServerGroups 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListServerGroups(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleQuery(c, func(ctx context.Context, req service.ServerGroupTreeQuery) ([]service.ServerGroupItem, error) {
		req.ProjectID = projectID
		return h.svc.ListServerGroupTree(ctx, req)
	})
}

// UpsertServerGroup 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ServerGroupUpsertRequest) (*service.ServerGroupItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertServerGroup(ctx, req)
	})
}

// UpdateServerGroup 更新对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) UpdateServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	groupID, err := parseUintParam(c, "groupId")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ServerGroupUpsertRequest) (*service.ServerGroupItem, error) {
		req.ProjectID = projectID
		req.ID = &groupID
		return h.svc.UpsertServerGroup(ctx, req)
	})
}

// DeleteServerGroup 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	groupID, err := parseUintParam(c, "groupId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteServerGroup(c.Request.Context(), projectID, groupID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListCloudAccounts 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListCloudAccounts(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleQuery(c, func(ctx context.Context, req service.CloudAccountListQuery) ([]service.CloudAccountItem, error) {
		req.ProjectID = projectID
		return h.svc.ListCloudAccounts(ctx, req)
	})
}

// UpsertCloudAccount 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.CloudAccountUpsertRequest) (*service.CloudAccountItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertCloudAccount(ctx, req)
	})
}

// UpdateCloudAccount 更新对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) UpdateCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.CloudAccountUpsertRequest) (*service.CloudAccountItem, error) {
		req.ProjectID = projectID
		req.ID = &accountID
		return h.svc.UpsertCloudAccount(ctx, req)
	})
}

// DeleteCloudAccount 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteCloudAccount(c.Request.Context(), projectID, accountID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// SyncCloudAccount 同步对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) SyncCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.SyncCloudAccount(c.Request.Context(), service.CloudSyncRequest{
		ProjectID: projectID,
		AccountID: accountID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// TestServer 测试对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) TestServer(c *gin.Context) {
	handleJSON(c, h.svc.TestServerConnectivity)
}

// BatchTestServers 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) BatchTestServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.BatchServerTestRequest) (*service.BatchServerTestResult, error) {
		req.ProjectID = projectID
		return h.svc.BatchTestServerConnectivity(ctx, req)
	})
}

// CloudServerAction 执行云服务器操作（改密/重启/关机）。
func (h *ProjectHandler) CloudServerAction(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.CloudServerActionRequest) (*service.CloudServerActionResult, error) {
		return h.svc.RunCloudServerAction(ctx, projectID, serverID, req)
	})
}

// SyncServers 同步对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) SyncServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ServerSyncRequest) (*service.ServerSyncResult, error) {
		req.ProjectID = projectID
		return h.svc.SyncProjectServers(ctx, req)
	})
}

// ImportServers 导入对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ImportServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("文件上传失败"))
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

// ExportServers 导出对应的 HTTP 接口处理逻辑。
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

// ServersImportTemplate 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) ServersImportTemplate(c *gin.Context) {
	f, err := h.svc.ServersImportTemplateExcel()
	if err != nil {
		response.Error(c, err)
		return
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := "servers-import-template.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ListServices 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListServices(c *gin.Context) {
	handleQuery(c, h.svc.ListServices)
}

// UpsertService 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertService(c *gin.Context) {
	handleJSON(c, h.svc.UpsertService)
}

// DeleteService 删除对应的 HTTP 接口处理逻辑。
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

// ListLogSources 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogSources(c *gin.Context) {
	handleQuery(c, func(ctx context.Context, q service.LogSourceListQuery) (gin.H, error) {
		req := &pb.ListLogSourcesRequest{
			ProjectId: uint64(q.ProjectID),
			Page:      &pb.PageRequest{Page: int32(q.Page), PageSize: int32(q.PageSize)},
		}
		if q.ServiceID != nil {
			req.HasServiceId = true
			req.ServiceId = uint64(*q.ServiceID)
		}
		resp, err := h.logSourceClient.ListLogSources(ctx, req)
		if err != nil {
			return nil, grpcToAppError(err)
		}
		out := make([]service.LogSourceItem, 0, len(resp.GetList()))
		for _, it := range resp.GetList() {
			out = append(out, service.LogSourceItem{
				ID:            uint(it.GetId()),
				ServiceID:     uint(it.GetServiceId()),
				LogType:       it.GetLogType(),
				Path:          it.GetPath(),
				Encoding:      stringPtr(it.GetEncoding()),
				Timezone:      stringPtr(it.GetTimezone()),
				MultilineRule: stringPtr(it.GetMultilineRule()),
				IncludeRegex:  stringPtr(it.GetIncludeRegex()),
				ExcludeRegex:  stringPtr(it.GetExcludeRegex()),
				Status:        int(it.GetStatus()),
				CreatedAt:     it.GetCreatedAt(),
			})
		}
		return gin.H{
			"list":      out,
			"total":     resp.GetPage().GetTotal(),
			"page":      resp.GetPage().GetPage(),
			"page_size": resp.GetPage().GetPageSize(),
		}, nil
	})
}

// UpsertLogSource 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertLogSource(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.LogSourceUpsertRequest) (service.LogSourceItem, error) {
		r := &pb.UpsertLogSourceRequest{
			ServiceId: uint64(req.ServiceID),
			LogType:   req.LogType,
			Path:      req.Path,
			Status:    int32(req.Status),
		}
		if req.ID != nil {
			r.Id = uint64(*req.ID)
		}
		if req.Encoding != nil {
			r.Encoding = *req.Encoding
		}
		if req.Timezone != nil {
			r.Timezone = *req.Timezone
		}
		if req.MultilineRule != nil {
			r.MultilineRule = *req.MultilineRule
		}
		if req.IncludeRegex != nil {
			r.IncludeRegex = *req.IncludeRegex
		}
		if req.ExcludeRegex != nil {
			r.ExcludeRegex = *req.ExcludeRegex
		}
		it, err := h.logSourceClient.UpsertLogSource(ctx, r)
		if err != nil {
			return service.LogSourceItem{}, grpcToAppError(err)
		}
		return service.LogSourceItem{
			ID:            uint(it.GetId()),
			ServiceID:     uint(it.GetServiceId()),
			LogType:       it.GetLogType(),
			Path:          it.GetPath(),
			Encoding:      stringPtr(it.GetEncoding()),
			Timezone:      stringPtr(it.GetTimezone()),
			MultilineRule: stringPtr(it.GetMultilineRule()),
			IncludeRegex:  stringPtr(it.GetIncludeRegex()),
			ExcludeRegex:  stringPtr(it.GetExcludeRegex()),
			Status:        int(it.GetStatus()),
			CreatedAt:     it.GetCreatedAt(),
		}, nil
	})
}

// DeleteLogSource 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteLogSource(c *gin.Context) {
	id, err := parseUintParam(c, "logSourceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if _, err := h.logSourceClient.DeleteLogSource(c.Request.Context(), &pb.DeleteLogSourceRequest{Id: uint64(id)}); err != nil {
		response.Error(c, grpcToAppError(err))
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// StreamLogs 处理对应的 HTTP 请求并返回统一响应。
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

	q, ok := bindQuery[service.LogStreamQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID

	streamKey := service.BuildLogStreamKey(q.ProjectID, q.ServerID, q.LogSourceID)
	var includeRe *regexp.Regexp
	if q.Include != nil && strings.TrimSpace(*q.Include) != "" {
		re, err := regexp.Compile(*q.Include)
		if err != nil {
			response.Error(c, apperror.BadRequest("包含正则表达式不合法"))
			return
		}
		includeRe = re
	}
	var excludeRe *regexp.Regexp
	if q.Exclude != nil && strings.TrimSpace(*q.Exclude) != "" {
		re, err := regexp.Compile(*q.Exclude)
		if err != nil {
			response.Error(c, apperror.BadRequest("排除正则表达式不合法"))
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

// ExportLogs 导出对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ExportLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogExportQuery](c)
	if !ok {
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

// ListLogFiles 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogFiles(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleQuery(c, func(ctx context.Context, q service.RemoteLogFileQuery) (gin.H, error) {
		q.ProjectID = projectID
		files, err := h.svc.ListRemoteLogFiles(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": files}, nil
	})
}

// ListLogUnits 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogUnits(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleQuery(c, func(ctx context.Context, q service.RemoteLogUnitQuery) (gin.H, error) {
		q.ProjectID = projectID
		list, err := h.svc.ListRemoteLogUnits(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// ListProjectMembers 项目成员列表。
func (h *ProjectHandler) ListProjectMembers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListProjectMembers(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// AddProjectMember 添加项目成员。
func (h *ProjectHandler) AddProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ProjectMemberAddRequest) (*service.ProjectMemberItem, error) {
		return h.svc.AddProjectMember(ctx, projectID, req)
	})
}

// UpdateProjectMember 更新成员角色。
func (h *ProjectHandler) UpdateProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	memberID, err := parseUintParam(c, "memberId")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.ProjectMemberUpdateRequest) (*service.ProjectMemberItem, error) {
		return h.svc.UpdateProjectMember(ctx, projectID, memberID, req)
	})
}

// RemoveProjectMember 移除项目成员。
func (h *ProjectHandler) RemoveProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	memberID, err := parseUintParam(c, "memberId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.RemoveProjectMember(c.Request.Context(), projectID, memberID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
