package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/repository"
)

type AgentDiscoveryService struct {
	repo       *repository.AgentDiscoveryRepository
	agentRepo  *repository.LogAgentRepository
	serverRepo *repository.ServerRepository
}

func NewAgentDiscoveryService(repo *repository.AgentDiscoveryRepository, agentRepo *repository.LogAgentRepository, serverRepo *repository.ServerRepository) *AgentDiscoveryService {
	return &AgentDiscoveryService{repo: repo, agentRepo: agentRepo, serverRepo: serverRepo}
}

type AgentDiscoveryItem struct {
	Kind  string         `json:"kind"`  // file/dir/unit
	Value string         `json:"value"` // path or unit name
	Extra map[string]any `json:"extra,omitempty"`
}

type AgentDiscoveryReportRequest struct {
	Token string               `json:"token" binding:"required"`
	Items []AgentDiscoveryItem `json:"items" binding:"required"`
}

type AgentDiscoveryReportResult struct {
	Accepted int `json:"accepted"`
}

func (s *AgentDiscoveryService) Report(ctx context.Context, req AgentDiscoveryReportRequest) (*AgentDiscoveryReportResult, error) {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, apperror.BadRequest("token is required")
	}
	agentSvc := NewLogAgentService(s.agentRepo, s.serverRepo, nil, "")
	agent, err := agentSvc.AuthenticateByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	items := make([]model.AgentDiscovery, 0, len(req.Items))
	now := time.Now()
	for _, it := range req.Items {
		kind := strings.ToLower(strings.TrimSpace(it.Kind))
		val := strings.TrimSpace(it.Value)
		if val == "" {
			continue
		}
		if kind != "file" && kind != "dir" && kind != "unit" {
			continue
		}
		var extraStr *string
		if len(it.Extra) > 0 {
			if b, e := json.Marshal(it.Extra); e == nil {
				x := string(b)
				extraStr = &x
			}
		}
		items = append(items, model.AgentDiscovery{
			ProjectID:   agent.ProjectID,
			ServerID:    agent.ServerID,
			Kind:        kind,
			Value:       val,
			Extra:       extraStr,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}
	if err := s.repo.UpsertMany(ctx, agent.ProjectID, agent.ServerID, items); err != nil {
		return nil, err
	}
	return &AgentDiscoveryReportResult{Accepted: len(items)}, nil
}

type AgentDiscoveryListQuery struct {
	ProjectID uint    `form:"project_id"`
	ServerID  uint    `form:"server_id" binding:"required"`
	Kind      *string `form:"kind"`
	Limit     int     `form:"limit"`
}

type AgentDiscoveryListItem struct {
	Kind       string  `json:"kind"`
	Value      string  `json:"value"`
	Extra      *string `json:"extra,omitempty"`
	LastSeenAt string  `json:"last_seen_at"`
}

func (s *AgentDiscoveryService) List(ctx context.Context, q AgentDiscoveryListQuery) ([]AgentDiscoveryListItem, error) {
	// Validate server belongs to project.
	sv, err := s.serverRepo.GetByID(ctx, q.ServerID)
	if err != nil {
		return nil, err
	}
	if sv.ProjectID != q.ProjectID {
		return nil, apperror.Forbidden("server not in project")
	}
	list, err := s.repo.List(ctx, q.ProjectID, q.ServerID, q.Kind, q.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]AgentDiscoveryListItem, 0, len(list))
	for _, it := range list {
		out = append(out, AgentDiscoveryListItem{
			Kind:       it.Kind,
			Value:      it.Value,
			Extra:      it.Extra,
			LastSeenAt: it.LastSeenAt.Format(time.RFC3339),
		})
	}
	return out, nil
}
