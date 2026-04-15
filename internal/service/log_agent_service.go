package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/repository"

	"gorm.io/gorm"
)

type LogAgentService struct {
	repo           *repository.LogAgentRepository
	serverRepo     *repository.ServerRepository
	logRepo        *repository.LogSourceRepository
	registerSecret string
}

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

func (s *LogAgentService) Register(ctx context.Context, req LogAgentRegisterRequest) (*LogAgentRegisterResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.BadRequest("name is required")
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("server not found")
		}
		return nil, err
	}
	if req.ProjectID > 0 && req.ProjectID != sv.ProjectID {
		return nil, apperror.BadRequest("project_id does not match server")
	}
	if sv.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("server is disabled")
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
	existing.Version = req.Version
	existing.TokenHash = tokenHash
	existing.Status = model.StatusEnabled
	if err := s.repo.Save(ctx, existing); err != nil {
		return nil, err
	}
	return &LogAgentRegisterResult{ProjectID: projectID, AgentID: existing.ID, Token: token, WSIngest: "/api/v1/agents/ws/ingest?token=<token>"}, nil
}

func (s *LogAgentService) PublicRegister(ctx context.Context, req LogAgentPublicRegisterRequest) (*LogAgentRegisterResult, error) {
	if s.registerSecret == "" {
		return nil, apperror.Forbidden("public agent registration is disabled")
	}
	if strings.TrimSpace(req.RegisterSecret) != s.registerSecret {
		return nil, apperror.Unauthorized("invalid register_secret")
	}
	return s.Register(ctx, LogAgentRegisterRequest{
		ProjectID: 0,
		ServerID:  req.ServerID,
		Name:      req.Name,
		Version:   req.Version,
	})
}

func (s *LogAgentService) AuthenticateByToken(ctx context.Context, token string) (*model.LogAgent, error) {
	if token == "" {
		return nil, apperror.Unauthorized("missing agent token")
	}
	it, err := s.repo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Unauthorized("invalid agent token")
		}
		return nil, err
	}
	return it, nil
}

func (s *LogAgentService) TouchSeen(ctx context.Context, id uint) {
	_ = s.repo.TouchSeen(ctx, id)
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
}

func (s *LogAgentService) Status(ctx context.Context, projectID, serverID, logSourceID uint) (*LogAgentStatusResult, error) {
	out := &LogAgentStatusResult{
		ServerID:    serverID,
		LogSourceID: logSourceID,
		ModeHint:    "agent",
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && agent != nil {
		out.Name = agent.Name
		out.Version = agent.Version
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

func (s *LogAgentService) Bootstrap(ctx context.Context, req AgentBootstrapRequest) (*AgentBootstrapResult, error) {
	name := req.AgentName
	if strings.TrimSpace(name) == "" {
		name = "log-agent"
	}
	ver := req.AgentVer
	if strings.TrimSpace(ver) == "" {
		ver = "v0.1.0"
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("server not found")
		}
		return nil, err
	}
	if sv.ProjectID != req.ProjectID {
		return nil, apperror.BadRequest("server not in project")
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
	run := fmt.Sprintf(
		"./log-agent --server-url %s --project-id %d --server-id %d --token %s",
		shellQuoteSingle(strings.TrimRight(req.PlatformURL, "/")),
		req.ProjectID,
		req.ServerID,
		shellQuoteSingle(reg.Token),
	)
	systemd := fmt.Sprintf(`[Unit]
Description=Go Permission Log Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/go-permission-system
ExecStart=/opt/go-permission-system/log-agent --server-url %s --project-id %d --server-id %d --token %s
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
`,
		shellQuoteSingle(strings.TrimRight(req.PlatformURL, "/")),
		req.ProjectID,
		req.ServerID,
		shellQuoteSingle(reg.Token),
	)
	return &AgentBootstrapResult{
		AgentID:        reg.AgentID,
		Token:          reg.Token,
		RunCommand:     run,
		SystemdService: systemd,
	}, nil
}

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

func (s *LogAgentService) RotateToken(ctx context.Context, req AgentBootstrapRequest) (*AgentBootstrapResult, error) {
	return s.Bootstrap(ctx, req)
}

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

func (b *logBroker) HasRecentPublisher(key string, within time.Duration) bool {
	b.mu.RLock()
	t, ok := b.lastPublish[key]
	b.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(t) <= within
}

var AgentLogBroker = newLogBroker()
