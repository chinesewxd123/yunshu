package handler

import (
	"context"

	pb "go-permission-system/internal/grpc/proto"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
)

type LogAgentHandler struct {
	svc         *service.LogAgentService
	agentClient pb.AgentRuntimeServiceClient
}

// NewLogAgentHandler 创建相关逻辑。
func NewLogAgentHandler(svc *service.LogAgentService, agentClient pb.AgentRuntimeServiceClient) *LogAgentHandler {
	return &LogAgentHandler{svc: svc, agentClient: agentClient}
}

// Register 注册对应的 HTTP 接口处理逻辑。
func (h *LogAgentHandler) Register(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.LogAgentRegisterRequest) (*service.LogAgentRegisterResult, error) {
		out, err := h.agentClient.Register(ctx, &pb.RegisterRequest{
			ProjectId: uint64(req.ProjectID),
			ServerId:  uint64(req.ServerID),
			Name:      req.Name,
			Version:   req.Version,
		})
		if err != nil {
			return nil, grpcToAppError(err)
		}
		return &service.LogAgentRegisterResult{
			ProjectID: uint(out.GetProjectId()),
			AgentID:   uint(out.GetAgentId()),
			Token:     out.GetToken(),
		}, nil
	})
}

// PublicRegister 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) PublicRegister(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.LogAgentPublicRegisterRequest) (*service.LogAgentRegisterResult, error) {
		out, err := h.agentClient.PublicRegister(ctx, &pb.PublicRegisterRequest{
			ServerId:       uint64(req.ServerID),
			Name:           req.Name,
			Version:        req.Version,
			RegisterSecret: req.RegisterSecret,
		})
		if err != nil {
			return nil, grpcToAppError(err)
		}
		return &service.LogAgentRegisterResult{
			ProjectID: uint(out.GetProjectId()),
			AgentID:   uint(out.GetAgentId()),
			Token:     out.GetToken(),
		}, nil
	})
}

// ReportHealth 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) ReportHealth(c *gin.Context) {
	handleJSONOK(c, gin.H{"message": "ok"}, func(ctx context.Context, req service.LogAgentHealthReportRequest) error {
		return h.svc.ReportHealthByToken(ctx, req)
	})
}

// RuntimeConfig 执行对应的 HTTP 接口处理逻辑。
func (h *LogAgentHandler) RuntimeConfig(c *gin.Context) {
	token := c.Query("token")
	data, err := h.agentClient.GetRuntimeConfig(c.Request.Context(), &pb.GetRuntimeConfigRequest{Token: token})
	if err != nil {
		response.Error(c, grpcToAppError(err))
		return
	}
	out := &service.AgentRuntimeConfigResult{
		ProjectID: uint(data.GetProjectId()),
		ServerID:  uint(data.GetServerId()),
		Sources:   make([]service.AgentRuntimeSource, 0, len(data.GetSources())),
	}
	for _, src := range data.GetSources() {
		out.Sources = append(out.Sources, service.AgentRuntimeSource{
			LogSourceID: uint(src.GetLogSourceId()),
			LogType:     src.GetLogType(),
			Path:        src.GetPath(),
		})
	}
	response.Success(c, out)
}

type agentStatusQuery struct {
	ServerID    uint `form:"server_id" binding:"required"`
	LogSourceID uint `form:"log_source_id"`
}

type agentListQuery struct {
	Keyword      string `form:"keyword"`
	HealthStatus string `form:"health_status"`
	Online       *bool  `form:"online"`
}

// List 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) List(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q agentListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	out, err := h.svc.ListByProject(c.Request.Context(), service.LogAgentListQuery{
		ProjectID:    projectID,
		Keyword:      q.Keyword,
		HealthStatus: q.HealthStatus,
		Online:       q.Online,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": out})
}

// BatchRefreshHeartbeat 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) BatchRefreshHeartbeat(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AgentBatchHeartbeatRefreshRequest) (*service.AgentBatchHeartbeatRefreshResult, error) {
		req.ProjectID = projectID
		return h.svc.BatchRefreshHeartbeat(ctx, req)
	})
}

// Status 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) Status(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q agentStatusQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	out, err := h.svc.Status(c.Request.Context(), projectID, q.ServerID, q.LogSourceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

// Bootstrap 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) Bootstrap(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AgentBootstrapRequest) (*service.AgentBootstrapResult, error) {
		req.ProjectID = projectID
		data, err := h.agentClient.Bootstrap(ctx, &pb.AgentBootstrapRequest{
			ProjectId:   uint64(req.ProjectID),
			ServerId:    uint64(req.ServerID),
			LogSourceId: uint64(req.LogSourceID),
			SourceType:  req.SourceType,
			Path:        req.Path,
			PlatformUrl: req.PlatformURL,
			AgentName:   req.AgentName,
			AgentVer:    req.AgentVer,
		})
		if err != nil {
			return nil, grpcToAppError(err)
		}
		return &service.AgentBootstrapResult{
			AgentID:        uint(data.GetAgentId()),
			Token:          data.GetToken(),
			RunCommand:     data.GetRunCommand(),
			SystemdService: data.GetSystemdService(),
		}, nil
	})
}

// RotateToken 处理对应的 HTTP 请求并返回统一响应。
func (h *LogAgentHandler) RotateToken(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, req service.AgentBootstrapRequest) (*service.AgentBootstrapResult, error) {
		req.ProjectID = projectID
		data, err := h.agentClient.RotateToken(ctx, &pb.AgentBootstrapRequest{
			ProjectId:   uint64(req.ProjectID),
			ServerId:    uint64(req.ServerID),
			LogSourceId: uint64(req.LogSourceID),
			SourceType:  req.SourceType,
			Path:        req.Path,
			PlatformUrl: req.PlatformURL,
			AgentName:   req.AgentName,
			AgentVer:    req.AgentVer,
		})
		if err != nil {
			return nil, grpcToAppError(err)
		}
		return &service.AgentBootstrapResult{
			AgentID:        uint(data.GetAgentId()),
			Token:          data.GetToken(),
			RunCommand:     data.GetRunCommand(),
			SystemdService: data.GetSystemdService(),
		}, nil
	})
}
