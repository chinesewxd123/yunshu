package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/logpath"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/repository"
)

const discoveryStaleRetention = 7 * 24 * time.Hour

type AgentDiscoveryService struct {
	repo       *repository.AgentDiscoveryRepository
	agentRepo  *repository.LogAgentRepository
	serverRepo *repository.ServerRepository
	logRepo    *repository.LogSourceRepository
}

// NewAgentDiscoveryService 创建相关逻辑。
func NewAgentDiscoveryService(
	repo *repository.AgentDiscoveryRepository,
	agentRepo *repository.LogAgentRepository,
	serverRepo *repository.ServerRepository,
	logRepo *repository.LogSourceRepository,
) *AgentDiscoveryService {
	return &AgentDiscoveryService{repo: repo, agentRepo: agentRepo, serverRepo: serverRepo, logRepo: logRepo}
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

// Report 执行对应的业务逻辑。
func (s *AgentDiscoveryService) Report(ctx context.Context, req AgentDiscoveryReportRequest) (*AgentDiscoveryReportResult, error) {
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg488a8dce8ef5)
	}
	agentSvc := NewLogAgentService(s.agentRepo, s.serverRepo, s.logRepo, "", nil)
	agent, err := agentSvc.AuthenticateByToken(ctx, token)
	if err != nil {
		return nil, svcerr.Pass(ctx, "agent.discovery", "Report", err)
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
		return nil, svcerr.Pass(ctx, "agent.discovery", "upsert", err, "server_id", agent.ServerID)
	}
	_ = s.repo.PruneStale(ctx, agent.ProjectID, agent.ServerID, now.Add(-discoveryStaleRetention))
	return &AgentDiscoveryReportResult{Accepted: len(items)}, nil
}

type AgentDiscoveryListQuery struct {
	ProjectID     uint   `form:"project_id"`
	ServerID      uint   `form:"server_id" binding:"required"`
	Kind          *string `form:"kind"`
	Limit         int    `form:"limit"`
	LogSourceID   uint   `form:"log_source_id"`
	UnmatchedOnly bool   `form:"unmatched_only"`
	Prefix        string `form:"prefix"`
	FreshHours    int    `form:"fresh_hours"`
}

type AgentDiscoveryListItem struct {
	Kind       string  `json:"kind"`
	Value      string  `json:"value"`
	Extra      *string `json:"extra,omitempty"`
	LastSeenAt string  `json:"last_seen_at"`
}

// List 查询列表相关的业务逻辑。
func (s *AgentDiscoveryService) List(ctx context.Context, q AgentDiscoveryListQuery) ([]AgentDiscoveryListItem, error) {
	sv, err := s.serverRepo.GetByID(ctx, q.ServerID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "agent.discovery", "List", err)
	}
	if sv.ProjectID != q.ProjectID {
		return nil, constants.ErrServerNotInProjectForbidden
	}

	freshHours := q.FreshHours
	if freshHours == 0 {
		freshHours = 24 * 7
	}
	var freshSince *time.Time
	if freshHours > 0 {
		t := time.Now().Add(-time.Duration(freshHours) * time.Hour)
		freshSince = &t
	}
	_ = s.repo.PruneStale(ctx, q.ProjectID, q.ServerID, time.Now().Add(-discoveryStaleRetention))

	kind := q.Kind
	if kind == nil {
		file := "file"
		kind = &file
	}
	list, err := s.repo.List(ctx, repository.AgentDiscoveryListFilter{
		ProjectID:  q.ProjectID,
		ServerID:   q.ServerID,
		Kind:       kind,
		Limit:      q.Limit,
		Prefix:     strings.TrimSpace(q.Prefix),
		FreshSince: freshSince,
	})
	if err != nil {
		return nil, svcerr.Pass(ctx, "agent.discovery", "List", err)
	}

	sources, _ := s.logRepo.ListByProjectAndServer(ctx, q.ProjectID, q.ServerID)
	var matchSourcePath string
	if q.LogSourceID > 0 {
		for _, src := range sources {
			if src.ID == q.LogSourceID {
				matchSourcePath = src.Path
				break
			}
		}
	}

	filtered := make([]model.AgentDiscovery, 0, len(list))
	for _, it := range list {
		if matchSourcePath != "" {
			if !logpath.PathMatchesSource(it.Value, matchSourcePath) {
				continue
			}
			filtered = append(filtered, it)
			continue
		}
		if q.UnmatchedOnly {
			matched := false
			for _, src := range sources {
				if logpath.PathMatchesSource(it.Value, src.Path) {
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}
		filtered = append(filtered, it)
	}

	out := make([]AgentDiscoveryListItem, 0, len(filtered))
	for _, it := range filtered {
		out = append(out, AgentDiscoveryListItem{
			Kind:       it.Kind,
			Value:      it.Value,
			Extra:      it.Extra,
			LastSeenAt: it.LastSeenAt.Format(time.RFC3339),
		})
	}
	return out, nil
}
