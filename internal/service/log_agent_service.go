package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	agentpkg "yunshu/internal/agent"
	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type LogAgentService struct {
	repo           *repository.LogAgentRepository
	serverRepo     *repository.ServerRepository
	logRepo        *repository.LogSourceRepository
	registerSecret string
}

// NewLogAgentService 创建相关逻辑。
func NewLogAgentService(repo *repository.LogAgentRepository, serverRepo *repository.ServerRepository, logRepo *repository.LogSourceRepository, registerSecret string) *LogAgentService {
	return &LogAgentService{repo: repo, serverRepo: serverRepo, logRepo: logRepo, registerSecret: strings.TrimSpace(registerSecret)}
}

type LogAgentRegisterRequest struct {
	ProjectID uint   `json:"project_id"`
	ServerID  uint   `json:"server_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Version   string `json:"version"`
}

type LogAgentPublicRegisterRequest struct {
	ServerID       uint   `json:"server_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Version        string `json:"version"`
	RegisterSecret string `json:"register_secret" binding:"required"`
}

type LogAgentRegisterResult struct {
	ProjectID uint   `json:"project_id"`
	AgentID   uint   `json:"agent_id"`
	Token     string `json:"token"`
	WSIngest  string `json:"ws_ingest"`
}

type LogAgentHealthReportRequest struct {
	Token           string `json:"token" binding:"required"`
	ListenPort      int    `json:"listen_port"`
	InstallProgress int    `json:"install_progress"`
	HealthStatus    string `json:"health_status"`
	LastError       string `json:"last_error"`
	Version         string `json:"version"`
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Register 注册相关的业务逻辑。
func (s *LogAgentService) Register(ctx context.Context, req LogAgentRegisterRequest) (*LogAgentRegisterResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.BadRequest("名称不能为空")
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("服务器不存在")
		}
		return nil, err
	}
	if req.ProjectID > 0 && req.ProjectID != sv.ProjectID {
		return nil, apperror.BadRequest("项目 ID 与服务器归属不匹配")
	}
	if sv.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("服务器已禁用")
	}
	projectID := sv.ProjectID

	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	tokenHash := hashToken(token)

	existing, err := s.repo.GetByServerID(ctx, req.ServerID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		it := &model.LogAgent{
			ProjectID: projectID,
			ServerID:  req.ServerID,
			Name:      strings.TrimSpace(req.Name),
			Version:   req.Version,
			TokenHash: tokenHash,
			Status:    model.StatusEnabled,
		}
		if err := s.repo.Create(ctx, it); err != nil {
			return nil, err
		}
		return &LogAgentRegisterResult{ProjectID: projectID, AgentID: it.ID, Token: token, WSIngest: "/api/v1/agents/ws/ingest?token=<token>"}, nil
	}
	existing.ProjectID = projectID
	existing.Name = strings.TrimSpace(req.Name)
	if v := strings.TrimSpace(req.Version); v != "" {
		existing.Version = v
	}
	existing.TokenHash = tokenHash
	existing.Status = model.StatusEnabled
	if err := s.repo.Save(ctx, existing); err != nil {
		return nil, err
	}
	return &LogAgentRegisterResult{ProjectID: projectID, AgentID: existing.ID, Token: token, WSIngest: "/api/v1/agents/ws/ingest?token=<token>"}, nil
}

// PublicRegister 执行对应的业务逻辑。
func (s *LogAgentService) PublicRegister(ctx context.Context, req LogAgentPublicRegisterRequest) (*LogAgentRegisterResult, error) {
	if s.registerSecret == "" {
		return nil, apperror.Forbidden("公共 Agent 注册已关闭")
	}
	if strings.TrimSpace(req.RegisterSecret) != s.registerSecret {
		return nil, apperror.Unauthorized("注册密钥无效")
	}
	return s.Register(ctx, LogAgentRegisterRequest{
		ProjectID: 0,
		ServerID:  req.ServerID,
		Name:      req.Name,
		Version:   req.Version,
	})
}

// AuthenticateByToken 执行对应的业务逻辑。
func (s *LogAgentService) AuthenticateByToken(ctx context.Context, token string) (*model.LogAgent, error) {
	if token == "" {
		return nil, apperror.Unauthorized("缺少 Agent 令牌")
	}
	it, err := s.repo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Unauthorized("Agent 令牌无效")
		}
		return nil, err
	}
	return it, nil
}

// TouchSeen 执行对应的业务逻辑。
func (s *LogAgentService) TouchSeen(ctx context.Context, id uint) {
	_ = s.repo.TouchSeen(ctx, id)
}

// TouchSeenByProjectServer refreshes agent heartbeat by project/server pair.
// Used by gRPC ingest stream where only project/server identifiers are available.
func (s *LogAgentService) TouchSeenByProjectServer(ctx context.Context, projectID, serverID uint) {
	it, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil || it == nil {
		return
	}
	_ = s.repo.TouchSeen(ctx, it.ID)
}

// ReportHealthByToken 上报 Agent 健康状态（对齐 go-ops 可观测字段）。
func (s *LogAgentService) ReportHealthByToken(ctx context.Context, req LogAgentHealthReportRequest) error {
	agent, err := s.AuthenticateByToken(ctx, strings.TrimSpace(req.Token))
	if err != nil {
		return err
	}
	now := time.Now()
	agent.LastSeenAt = &now
	// 与 Agent 上报一致：0 表示未在本机监听端口（仅出站）
	agent.ListenPort = req.ListenPort
	if req.InstallProgress < 0 {
		req.InstallProgress = 0
	}
	if req.InstallProgress > 100 {
		req.InstallProgress = 100
	}
	agent.InstallProgress = req.InstallProgress
	if v := strings.TrimSpace(req.HealthStatus); v != "" {
		agent.HealthStatus = v
	}
	agent.LastError = strings.TrimSpace(req.LastError)
	if v := strings.TrimSpace(req.Version); v != "" {
		cur := strings.TrimSpace(agent.Version)
		// 进程默认版本不应覆盖部署/登记时写入的版本（Bootstrap 曾未带 --version，导致一直上报 v0.1.0）。
		if v != agentpkg.DefaultVersion || cur == "" {
			agent.Version = v
		}
	}
	return s.repo.Save(ctx, agent)
}

type LogAgentStatusResult struct {
	ServerID         uint    `json:"server_id"`
	LogSourceID      uint    `json:"log_source_id"`
	AgentID          *uint   `json:"agent_id,omitempty"`
	Name             string  `json:"name,omitempty"`
	Version          string  `json:"version,omitempty"`
	LastSeenAt       *string `json:"last_seen_at,omitempty"`
	Online           bool    `json:"online"`
	RecentPublishing bool    `json:"recent_publishing"`
	ModeHint         string  `json:"mode_hint"`
	ListenPort       int     `json:"listen_port"`
	InstallProgress  int     `json:"install_progress"`
	HealthStatus     string  `json:"health_status"`
	LastError        string  `json:"last_error,omitempty"`
}

type LogAgentListQuery struct {
	ProjectID    uint   `form:"project_id" binding:"required"`
	Keyword      string `form:"keyword"`
	HealthStatus string `form:"health_status"`
	Online       *bool  `form:"online"`
}

type LogAgentListItem struct {
	ServerID         uint    `json:"server_id"`
	ServerName       string  `json:"server_name"`
	ServerHost       string  `json:"server_host"`
	ProjectName      string  `json:"project_name,omitempty"`
	AgentID          *uint   `json:"agent_id,omitempty"`
	Name             string  `json:"name,omitempty"`
	Version          string  `json:"version,omitempty"`
	LastSeenAt       *string `json:"last_seen_at,omitempty"`
	Online           bool    `json:"online"`
	ListenPort       int     `json:"listen_port"`
	InstallProgress  int     `json:"install_progress"`
	HealthStatus     string  `json:"health_status"`
	LastError        string  `json:"last_error,omitempty"`
	RecentPublishing bool    `json:"recent_publishing"`
}

func (s *LogAgentService) ListByProject(ctx context.Context, q LogAgentListQuery) ([]LogAgentListItem, error) {
	servers, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: q.ProjectID,
		Keyword:   strings.TrimSpace(q.Keyword),
		Page:      1,
		PageSize:  10000,
	})
	if err != nil {
		return nil, err
	}
	projectName, _ := s.serverRepo.ProjectNameByID(ctx, q.ProjectID)
	out := make([]LogAgentListItem, 0, len(servers))
	for _, sv := range servers {
		item, err := s.buildListItem(ctx, q.ProjectID, sv)
		if err != nil {
			return nil, err
		}
		item.ProjectName = projectName
		if q.Online != nil && item.Online != *q.Online {
			continue
		}
		if hs := strings.TrimSpace(q.HealthStatus); hs != "" && !strings.EqualFold(item.HealthStatus, hs) {
			continue
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *LogAgentService) buildListItem(ctx context.Context, projectID uint, sv model.Server) (*LogAgentListItem, error) {
	item := &LogAgentListItem{
		ServerID:        sv.ID,
		ServerName:      sv.Name,
		ServerHost:      sv.Host,
		ListenPort:      0,
		HealthStatus:    "unknown",
		Online:          false,
		LastError:       "",
		InstallProgress: 0,
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, sv.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && agent != nil {
		item.Name = agent.Name
		item.Version = agent.Version
		item.ListenPort = agent.ListenPort
		item.InstallProgress = agent.InstallProgress
		item.HealthStatus = agent.HealthStatus
		item.LastError = agent.LastError
		id := agent.ID
		item.AgentID = &id
		if agent.LastSeenAt != nil {
			x := agent.LastSeenAt.Format(time.RFC3339)
			item.LastSeenAt = &x
			item.Online = time.Since(*agent.LastSeenAt) <= 90*time.Second
		}
	}
	sources, err := s.logRepo.ListByProjectAndServer(ctx, projectID, sv.ID)
	if err == nil {
		for _, src := range sources {
			key := BuildLogStreamKey(projectID, sv.ID, src.ID)
			if AgentLogBroker.HasRecentPublisher(key, 30*time.Second) {
				item.RecentPublishing = true
				break
			}
		}
	}
	return item, nil
}

type AgentBatchHeartbeatRefreshRequest struct {
	ProjectID uint   `json:"project_id"`
	ServerIDs []uint `json:"server_ids"`
}

type AgentBatchHeartbeatRefreshResult struct {
	Refreshed int                `json:"refreshed"`
	List      []LogAgentListItem `json:"list"`
}

func (s *LogAgentService) BatchRefreshHeartbeat(ctx context.Context, req AgentBatchHeartbeatRefreshRequest) (*AgentBatchHeartbeatRefreshResult, error) {
	if req.ProjectID == 0 {
		return nil, apperror.BadRequest("project_id 不能为空")
	}
	serverIDSet := map[uint]struct{}{}
	for _, id := range req.ServerIDs {
		if id > 0 {
			serverIDSet[id] = struct{}{}
		}
	}
	servers, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: req.ProjectID,
		Page:      1,
		PageSize:  10000,
	})
	if err != nil {
		return nil, err
	}
	projectName, _ := s.serverRepo.ProjectNameByID(ctx, req.ProjectID)
	list := make([]LogAgentListItem, 0)
	for _, sv := range servers {
		if len(serverIDSet) > 0 {
			if _, ok := serverIDSet[sv.ID]; !ok {
				continue
			}
		}
		item, err := s.buildListItem(ctx, req.ProjectID, sv)
		if err != nil {
			return nil, err
		}
		item.ProjectName = projectName
		if item.RecentPublishing && item.AgentID != nil {
			_ = s.repo.TouchSeen(ctx, *item.AgentID)
			agent, aerr := s.repo.GetByProjectAndServer(ctx, req.ProjectID, sv.ID)
			if aerr == nil && agent.LastSeenAt != nil {
				x := agent.LastSeenAt.Format(time.RFC3339)
				item.LastSeenAt = &x
				item.Online = true
			}
		}
		list = append(list, *item)
	}
	return &AgentBatchHeartbeatRefreshResult{
		Refreshed: len(list),
		List:      list,
	}, nil
}

// Status 执行对应的业务逻辑。
func (s *LogAgentService) Status(ctx context.Context, projectID, serverID, logSourceID uint) (*LogAgentStatusResult, error) {
	out := &LogAgentStatusResult{
		ServerID:     serverID,
		LogSourceID:  logSourceID,
		ModeHint:     "agent",
		ListenPort:   0,
		HealthStatus: "unknown",
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && agent != nil {
		out.Name = agent.Name
		out.Version = agent.Version
		out.ListenPort = agent.ListenPort
		out.InstallProgress = agent.InstallProgress
		out.HealthStatus = agent.HealthStatus
		out.LastError = agent.LastError
		id := agent.ID
		out.AgentID = &id
		if agent.LastSeenAt != nil {
			x := agent.LastSeenAt.Format(time.RFC3339)
			out.LastSeenAt = &x
			out.Online = time.Since(*agent.LastSeenAt) <= 90*time.Second
		}
	}
	if logSourceID > 0 {
		key := BuildLogStreamKey(projectID, serverID, logSourceID)
		out.RecentPublishing = AgentLogBroker.HasRecentPublisher(key, 30*time.Second)
	}
	return out, nil
}

type AgentBootstrapRequest struct {
	ProjectID   uint   `json:"project_id"`
	ServerID    uint   `json:"server_id" binding:"required"`
	LogSourceID uint   `json:"log_source_id"`
	SourceType  string `json:"source_type"`
	Path        string `json:"path"`
	PlatformURL string `json:"platform_url" binding:"required"`
	AgentName   string `json:"agent_name"`
	AgentVer    string `json:"agent_version"`
}

type AgentBootstrapResult struct {
	AgentID        uint   `json:"agent_id"`
	Token          string `json:"token"`
	RunCommand     string `json:"run_command"`
	SystemdService string `json:"systemd_service"`
}

type AgentRuntimeSource struct {
	LogSourceID uint   `json:"log_source_id"`
	LogType     string `json:"log_type"`
	Path        string `json:"path"`
}

type AgentRuntimeConfigResult struct {
	ProjectID uint                 `json:"project_id"`
	ServerID  uint                 `json:"server_id"`
	Sources   []AgentRuntimeSource `json:"sources"`
}

type AgentLogEvent struct {
	Line     string `json:"line"`
	FilePath string `json:"file_path,omitempty"`
}

func shellQuoteSingle(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func normalizeGrpcServerTarget(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "127.0.0.1:18080"
	}
	u, err := neturl.Parse(v)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		// already host:port style
		return strings.TrimRight(v, "/")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return "127.0.0.1:18080"
	}
	port := strings.TrimSpace(u.Port())
	// UI still tends to pass frontend/backend HTTP url; map to grpc default port.
	if port == "" || port == "5173" || port == "8080" {
		port = "18080"
	}
	return host + ":" + port
}

// Bootstrap 执行对应的业务逻辑。
func (s *LogAgentService) Bootstrap(ctx context.Context, req AgentBootstrapRequest) (*AgentBootstrapResult, error) {
	name := req.AgentName
	if strings.TrimSpace(name) == "" {
		name = "log-agent"
	}
	ver := req.AgentVer
	if strings.TrimSpace(ver) == "" {
		ver = agentpkg.DefaultVersion
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("服务器不存在")
		}
		return nil, err
	}
	if sv.ProjectID != req.ProjectID {
		return nil, apperror.BadRequest("服务器不在当前项目中")
	}
	reg, err := s.Register(ctx, LogAgentRegisterRequest{
		ProjectID: req.ProjectID,
		ServerID:  req.ServerID,
		Name:      name,
		Version:   ver,
	})
	if err != nil {
		return nil, err
	}
	grpcTarget := normalizeGrpcServerTarget(req.PlatformURL)
	run := fmt.Sprintf(
		"./log-agent --grpc-server %s --server-id %d --token %s --version %s --enable-runtime-pull=true --enable-discovery=true --enable-health-report=true --enable-fallback=false",
		shellQuoteSingle(grpcTarget),
		req.ServerID,
		shellQuoteSingle(reg.Token),
		shellQuoteSingle(ver),
	)
	systemd := fmt.Sprintf(`[Unit]
Description=Go Permission Log Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/yunshu
ExecStart=/opt/yunshu/log-agent --grpc-server %s --server-id %d --token %s --version %s --enable-runtime-pull=true --enable-discovery=true --enable-health-report=true --enable-fallback=false
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
`,
		shellQuoteSingle(grpcTarget),
		req.ServerID,
		shellQuoteSingle(reg.Token),
		shellQuoteSingle(ver),
	)
	return &AgentBootstrapResult{
		AgentID:        reg.AgentID,
		Token:          reg.Token,
		RunCommand:     run,
		SystemdService: systemd,
	}, nil
}

// RuntimeConfigByToken 执行相关的业务逻辑。
func (s *LogAgentService) RuntimeConfigByToken(ctx context.Context, token string) (*AgentRuntimeConfigResult, error) {
	agent, err := s.AuthenticateByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	sources, err := s.logRepo.ListByProjectAndServer(ctx, agent.ProjectID, agent.ServerID)
	if err != nil {
		return nil, err
	}
	out := make([]AgentRuntimeSource, 0, len(sources))
	for _, it := range sources {
		out = append(out, AgentRuntimeSource{
			LogSourceID: it.ID,
			LogType:     it.LogType,
			Path:        it.Path,
		})
	}
	return &AgentRuntimeConfigResult{
		ProjectID: agent.ProjectID,
		ServerID:  agent.ServerID,
		Sources:   out,
	}, nil
}

// RotateToken 执行对应的业务逻辑。
func (s *LogAgentService) RotateToken(ctx context.Context, req AgentBootstrapRequest) (*AgentBootstrapResult, error) {
	return s.Bootstrap(ctx, req)
}

// BuildLogStreamKey 构建相关逻辑。
func BuildLogStreamKey(projectID, serverID, logSourceID uint) string {
	return fmt.Sprintf("%d:%d:%d", projectID, serverID, logSourceID)
}

type logBroker struct {
	mu          sync.RWMutex
	subs        map[string]map[chan AgentLogEvent]struct{}
	lastPublish map[string]time.Time
	history     map[string][]AgentLogEvent
}

const maxLogHistoryPerStream = 5000

func newLogBroker() *logBroker {
	return &logBroker{
		subs:        map[string]map[chan AgentLogEvent]struct{}{},
		lastPublish: map[string]time.Time{},
		history:     map[string][]AgentLogEvent{},
	}
}

// Publish 的功能实现。
func (b *logBroker) Publish(key string, event AgentLogEvent) {
	if strings.TrimSpace(event.Line) == "" {
		return
	}
	b.mu.Lock()
	b.lastPublish[key] = time.Now()
	h := append(b.history[key], event)
	if len(h) > maxLogHistoryPerStream {
		h = append([]AgentLogEvent(nil), h[len(h)-maxLogHistoryPerStream:]...)
	}
	b.history[key] = h
	targets := b.subs[key]
	b.mu.Unlock()
	for ch := range targets {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe 的功能实现。
func (b *logBroker) Subscribe(key string, replayLines int) (<-chan AgentLogEvent, func()) {
	if replayLines < 0 {
		replayLines = 0
	}
	bufSize := 512
	if replayLines > bufSize {
		bufSize = replayLines + 64
	}
	ch := make(chan AgentLogEvent, bufSize)
	b.mu.Lock()
	if _, ok := b.subs[key]; !ok {
		b.subs[key] = map[chan AgentLogEvent]struct{}{}
	}
	b.subs[key][ch] = struct{}{}
	history := b.history[key]
	start := 0
	if replayLines > 0 && len(history) > replayLines {
		start = len(history) - replayLines
	}
	snapshot := append([]AgentLogEvent(nil), history[start:]...)
	b.mu.Unlock()
	for _, it := range snapshot {
		select {
		case ch <- it:
		default:
			// keep latest when replay burst exceeds buffer
		}
	}
	cancel := func() {
		b.mu.Lock()
		if m, ok := b.subs[key]; ok {
			delete(m, ch)
			if len(m) == 0 {
				delete(b.subs, key)
			}
		}
		b.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// HasRecentPublisher 的功能实现。
func (b *logBroker) HasRecentPublisher(key string, within time.Duration) bool {
	b.mu.RLock()
	t, ok := b.lastPublish[key]
	b.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(t) <= within
}

// Snapshot 返回指定 stream key 的历史快照（按时间升序，最多 maxLines 条）。
func (b *logBroker) Snapshot(key string, maxLines int) []AgentLogEvent {
	if maxLines <= 0 {
		maxLines = 2000
	}
	if maxLines > maxLogHistoryPerStream {
		maxLines = maxLogHistoryPerStream
	}
	b.mu.RLock()
	history := b.history[key]
	start := 0
	if len(history) > maxLines {
		start = len(history) - maxLines
	}
	snapshot := append([]AgentLogEvent(nil), history[start:]...)
	b.mu.RUnlock()
	return snapshot
}

var AgentLogBroker = newLogBroker()
