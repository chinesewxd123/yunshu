package server

import (
	"context"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"
	"yunshu/internal/service"
)

type LogPlatformServer struct {
	pb.UnimplementedProjectServerServiceServer
	pb.UnimplementedLogSourceServiceServer
	pb.UnimplementedAgentRuntimeServiceServer

	projectSvc   *service.ProjectMgmtService
	agentSvc     *service.LogAgentService
	discoverySvc *service.AgentDiscoveryService
}

func NewLogPlatformServer(projectSvc *service.ProjectMgmtService, agentSvc *service.LogAgentService, discoverySvc *service.AgentDiscoveryService) *LogPlatformServer {
	return &LogPlatformServer{
		projectSvc:   projectSvc,
		agentSvc:     agentSvc,
		discoverySvc: discoverySvc,
	}
}

func (s *LogPlatformServer) ListServers(ctx context.Context, req *pb.ListServersRequest) (*pb.ListServersResponse, error) {
	out, err := s.projectSvc.ListServers(ctx, service.ServerListQuery{
		ProjectID: uint(req.GetProjectId()),
		Keyword:   req.GetKeyword(),
		Page:      int(req.GetPage().GetPage()),
		PageSize:  int(req.GetPage().GetPageSize()),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	resp := &pb.ListServersResponse{
		List: make([]*pb.ServerItem, 0, len(out.List)),
		Page: &pb.PageResult{Total: out.Total, Page: int32(out.Page), PageSize: int32(out.PageSize)},
	}
	for _, it := range out.List {
		resp.List = append(resp.List, &pb.ServerItem{
			Id:            uint64(it.ID),
			ProjectId:     uint64(it.ProjectID),
			Name:          it.Name,
			Host:          it.Host,
			Port:          int32(it.Port),
			OsType:        it.OSType,
			OsArch:        it.OSArch,
			Tags:          it.Tags,
			LastTestAt:    derefString(it.LastTestAt),
			LastTestError: derefString(it.LastTestErr),
			CreatedAt:     it.CreatedAt,
			LastSeenAt:    derefString(it.LastSeenAt),
			Status:        int32(it.Status),
		})
	}
	return resp, nil
}

func (s *LogPlatformServer) GetServer(ctx context.Context, req *pb.GetServerRequest) (*pb.ServerItem, error) {
	out, err := s.projectSvc.GetServer(ctx, uint(req.GetId()))
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.ServerItem{
		Id:            uint64(out.ID),
		ProjectId:     uint64(out.ProjectID),
		Name:          out.Name,
		Host:          out.Host,
		Port:          int32(out.Port),
		OsType:        out.OSType,
		OsArch:        out.OSArch,
		Tags:          out.Tags,
		LastTestAt:    derefString(out.LastTestAt),
		LastTestError: derefString(out.LastTestErr),
		CreatedAt:     out.CreatedAt,
		LastSeenAt:    derefString(out.LastSeenAt),
		Status:        int32(out.Status),
	}, nil
}

func (s *LogPlatformServer) UpsertServer(ctx context.Context, req *pb.UpsertServerRequest) (*pb.ServerItem, error) {
	var id *uint
	if req.GetId() > 0 {
		v := uint(req.GetId())
		id = &v
	}
	var password, privateKey, passphrase *string
	if req.GetPassword() != "" {
		v := req.GetPassword()
		password = &v
	}
	if req.GetPrivateKey() != "" {
		v := req.GetPrivateKey()
		privateKey = &v
	}
	if req.GetPassphrase() != "" {
		v := req.GetPassphrase()
		passphrase = &v
	}
	item, err := s.projectSvc.UpsertServer(ctx, service.ServerUpsertRequest{
		ID:         id,
		ProjectID:  uint(req.GetProjectId()),
		Name:       req.GetName(),
		Host:       req.GetHost(),
		Port:       int(req.GetPort()),
		OSType:     req.GetOsType(),
		Tags:       req.GetTags(),
		Status:     int(req.GetStatus()),
		AuthType:   req.GetAuthType(),
		Username:   req.GetUsername(),
		Password:   password,
		PrivateKey: privateKey,
		Passphrase: passphrase,
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.ServerItem{
		Id:            uint64(item.ID),
		ProjectId:     uint64(item.ProjectID),
		Name:          item.Name,
		Host:          item.Host,
		Port:          int32(item.Port),
		OsType:        item.OSType,
		OsArch:        item.OSArch,
		Tags:          item.Tags,
		LastTestAt:    derefString(item.LastTestAt),
		LastTestError: derefString(item.LastTestErr),
		CreatedAt:     item.CreatedAt,
		LastSeenAt:    derefString(item.LastSeenAt),
		Status:        int32(item.Status),
	}, nil
}

func (s *LogPlatformServer) DeleteServer(ctx context.Context, req *pb.DeleteServerRequest) (*pb.DeleteServerResponse, error) {
	if err := s.projectSvc.DeleteServer(ctx, uint(req.GetId())); err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.DeleteServerResponse{Deleted: true}, nil
}

func (s *LogPlatformServer) ListLogSources(ctx context.Context, req *pb.ListLogSourcesRequest) (*pb.ListLogSourcesResponse, error) {
	var serviceID *uint
	if req.GetHasServiceId() {
		v := uint(req.GetServiceId())
		serviceID = &v
	}
	out, err := s.projectSvc.ListLogSources(ctx, service.LogSourceListQuery{
		ProjectID: uint(req.GetProjectId()),
		ServiceID: serviceID,
		Page:      int(req.GetPage().GetPage()),
		PageSize:  int(req.GetPage().GetPageSize()),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	resp := &pb.ListLogSourcesResponse{
		List: make([]*pb.LogSourceItem, 0, len(out.List)),
		Page: &pb.PageResult{Total: out.Total, Page: int32(out.Page), PageSize: int32(out.PageSize)},
	}
	for _, it := range out.List {
		resp.List = append(resp.List, &pb.LogSourceItem{
			Id:            uint64(it.ID),
			ServiceId:     uint64(it.ServiceID),
			LogType:       it.LogType,
			Path:          it.Path,
			Encoding:      derefString(it.Encoding),
			Timezone:      derefString(it.Timezone),
			MultilineRule: derefString(it.MultilineRule),
			IncludeRegex:  derefString(it.IncludeRegex),
			ExcludeRegex:  derefString(it.ExcludeRegex),
			Status:        int32(it.Status),
			CreatedAt:     it.CreatedAt,
		})
	}
	return resp, nil
}

func (s *LogPlatformServer) UpsertLogSource(ctx context.Context, req *pb.UpsertLogSourceRequest) (*pb.LogSourceItem, error) {
	var id *uint
	if req.GetId() > 0 {
		v := uint(req.GetId())
		id = &v
	}
	item, err := s.projectSvc.UpsertLogSource(ctx, service.LogSourceUpsertRequest{
		ID:            id,
		ServiceID:     uint(req.GetServiceId()),
		LogType:       req.GetLogType(),
		Path:          req.GetPath(),
		Encoding:      stringPtr(req.GetEncoding()),
		Timezone:      stringPtr(req.GetTimezone()),
		MultilineRule: stringPtr(req.GetMultilineRule()),
		IncludeRegex:  stringPtr(req.GetIncludeRegex()),
		ExcludeRegex:  stringPtr(req.GetExcludeRegex()),
		Status:        int(req.GetStatus()),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.LogSourceItem{
		Id:            uint64(item.ID),
		ServiceId:     uint64(item.ServiceID),
		LogType:       item.LogType,
		Path:          item.Path,
		Encoding:      derefString(item.Encoding),
		Timezone:      derefString(item.Timezone),
		MultilineRule: derefString(item.MultilineRule),
		IncludeRegex:  derefString(item.IncludeRegex),
		ExcludeRegex:  derefString(item.ExcludeRegex),
		Status:        int32(item.Status),
		CreatedAt:     item.CreatedAt,
	}, nil
}

func (s *LogPlatformServer) DeleteLogSource(ctx context.Context, req *pb.DeleteLogSourceRequest) (*pb.DeleteLogSourceResponse, error) {
	if err := s.projectSvc.DeleteLogSource(ctx, uint(req.GetId())); err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.DeleteLogSourceResponse{Deleted: true}, nil
}

func (s *LogPlatformServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	out, err := s.agentSvc.Register(ctx, service.LogAgentRegisterRequest{
		ProjectID: uint(req.GetProjectId()),
		ServerID:  uint(req.GetServerId()),
		Name:      req.GetName(),
		Version:   req.GetVersion(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.RegisterResponse{ProjectId: uint64(out.ProjectID), AgentId: uint64(out.AgentID), Token: out.Token}, nil
}

func (s *LogPlatformServer) PublicRegister(ctx context.Context, req *pb.PublicRegisterRequest) (*pb.RegisterResponse, error) {
	out, err := s.agentSvc.PublicRegister(ctx, service.LogAgentPublicRegisterRequest{
		ServerID:       uint(req.GetServerId()),
		Name:           req.GetName(),
		Version:        req.GetVersion(),
		RegisterSecret: req.GetRegisterSecret(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.RegisterResponse{ProjectId: uint64(out.ProjectID), AgentId: uint64(out.AgentID), Token: out.Token}, nil
}

func (s *LogPlatformServer) GetRuntimeConfig(ctx context.Context, req *pb.GetRuntimeConfigRequest) (*pb.GetRuntimeConfigResponse, error) {
	out, err := s.agentSvc.RuntimeConfigByToken(ctx, req.GetToken())
	if err != nil {
		return nil, toStatusErr(err)
	}
	resp := &pb.GetRuntimeConfigResponse{
		ProjectId: uint64(out.ProjectID),
		ServerId:  uint64(out.ServerID),
		Sources:   make([]*pb.RuntimeSource, 0, len(out.Sources)),
	}
	for _, src := range out.Sources {
		resp.Sources = append(resp.Sources, &pb.RuntimeSource{
			LogSourceId: uint64(src.LogSourceID),
			LogType:     src.LogType,
			Path:        src.Path,
		})
	}
	return resp, nil
}

func (s *LogPlatformServer) Status(ctx context.Context, req *pb.AgentStatusRequest) (*pb.AgentStatusResponse, error) {
	out, err := s.agentSvc.Status(ctx, uint(req.GetProjectId()), uint(req.GetServerId()), uint(req.GetLogSourceId()))
	if err != nil {
		return nil, toStatusErr(err)
	}
	resp := &pb.AgentStatusResponse{
		ServerId:         uint64(out.ServerID),
		LogSourceId:      uint64(out.LogSourceID),
		Name:             out.Name,
		Version:          out.Version,
		LastSeenAt:       derefString(out.LastSeenAt),
		Online:           out.Online,
		RecentPublishing: out.RecentPublishing,
		ModeHint:         out.ModeHint,
	}
	if out.AgentID != nil {
		resp.AgentId = uint64(*out.AgentID)
		resp.HasAgentId = true
	}
	return resp, nil
}

func (s *LogPlatformServer) Bootstrap(ctx context.Context, req *pb.AgentBootstrapRequest) (*pb.AgentBootstrapResponse, error) {
	out, err := s.agentSvc.Bootstrap(ctx, service.AgentBootstrapRequest{
		ProjectID:   uint(req.GetProjectId()),
		ServerID:    uint(req.GetServerId()),
		LogSourceID: uint(req.GetLogSourceId()),
		SourceType:  req.GetSourceType(),
		Path:        req.GetPath(),
		PlatformURL: req.GetPlatformUrl(),
		AgentName:   req.GetAgentName(),
		AgentVer:    req.GetAgentVer(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.AgentBootstrapResponse{
		AgentId:        uint64(out.AgentID),
		Token:          out.Token,
		RunCommand:     out.RunCommand,
		SystemdService: out.SystemdService,
	}, nil
}

func (s *LogPlatformServer) RotateToken(ctx context.Context, req *pb.AgentBootstrapRequest) (*pb.AgentBootstrapResponse, error) {
	out, err := s.agentSvc.RotateToken(ctx, service.AgentBootstrapRequest{
		ProjectID:   uint(req.GetProjectId()),
		ServerID:    uint(req.GetServerId()),
		LogSourceID: uint(req.GetLogSourceId()),
		SourceType:  req.GetSourceType(),
		Path:        req.GetPath(),
		PlatformURL: req.GetPlatformUrl(),
		AgentName:   req.GetAgentName(),
		AgentVer:    req.GetAgentVer(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.AgentBootstrapResponse{
		AgentId:        uint64(out.AgentID),
		Token:          out.Token,
		RunCommand:     out.RunCommand,
		SystemdService: out.SystemdService,
	}, nil
}

func (s *LogPlatformServer) ReportDiscovery(ctx context.Context, req *pb.ReportDiscoveryRequest) (*pb.ReportDiscoveryResponse, error) {
	items := make([]service.AgentDiscoveryItem, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		extra := map[string]any{}
		for k, v := range it.GetExtra() {
			extra[k] = v
		}
		items = append(items, service.AgentDiscoveryItem{Kind: it.GetKind(), Value: it.GetValue(), Extra: extra})
	}
	out, err := s.discoverySvc.Report(ctx, service.AgentDiscoveryReportRequest{
		Token: req.GetToken(),
		Items: items,
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return &pb.ReportDiscoveryResponse{Accepted: int32(out.Accepted)}, nil
}

func (s *LogPlatformServer) ListDiscovery(ctx context.Context, req *pb.ListDiscoveryRequest) (*pb.ListDiscoveryResponse, error) {
	var kind *string
	if strings.TrimSpace(req.GetKind()) != "" {
		v := req.GetKind()
		kind = &v
	}
	out, err := s.discoverySvc.List(ctx, service.AgentDiscoveryListQuery{
		ProjectID: uint(req.GetProjectId()),
		ServerID:  uint(req.GetServerId()),
		Kind:      kind,
		Limit:     int(req.GetLimit()),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	resp := &pb.ListDiscoveryResponse{List: make([]*pb.AgentDiscoveryItem, 0, len(out))}
	for _, it := range out {
		resp.List = append(resp.List, &pb.AgentDiscoveryItem{
			Kind:       it.Kind,
			Value:      it.Value,
			LastSeenAt: it.LastSeenAt,
		})
	}
	return resp, nil
}

func (s *LogPlatformServer) IngestLogs(stream pb.AgentRuntimeService_IngestLogsServer) error {
	lastSeenTouch := time.Time{}
	touchSeen := func(projectID, serverID uint) {
		now := time.Now()
		// Avoid touching DB on every batch; 30s is enough for 90s online window.
		if !lastSeenTouch.IsZero() && now.Sub(lastSeenTouch) < 30*time.Second {
			return
		}
		s.agentSvc.TouchSeenByProjectServer(stream.Context(), projectID, serverID)
		lastSeenTouch = now
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			return nil
		}
		projectID := uint(msg.GetProjectId())
		serverID := uint(msg.GetServerId())
		touchSeen(projectID, serverID)
		key := service.BuildLogStreamKey(projectID, serverID, uint(msg.GetLogSourceId()))
		for _, it := range msg.GetEntries() {
			if strings.TrimSpace(it.GetLine()) == "" {
				continue
			}
			service.AgentLogBroker.Publish(key, service.AgentLogEvent{
				Line:     it.GetLine(),
				FilePath: it.GetFilePath(),
			})
		}
		if msg.GetSeq() > 0 {
			if err := stream.Send(&pb.IngestLogsResponse{Seq: msg.GetSeq(), TsUnixMs: time.Now().UnixMilli()}); err != nil {
				return nil
			}
		}
	}
}

func stringPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
