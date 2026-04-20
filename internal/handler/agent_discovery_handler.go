package handler

import (
	"context"

	pb "yunshu/internal/grpc/proto"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type AgentDiscoveryHandler struct {
	svc         *service.AgentDiscoveryService
	agentClient pb.AgentRuntimeServiceClient
}

// NewAgentDiscoveryHandler 创建相关逻辑。
func NewAgentDiscoveryHandler(svc *service.AgentDiscoveryService, agentClient pb.AgentRuntimeServiceClient) *AgentDiscoveryHandler {
	return &AgentDiscoveryHandler{svc: svc, agentClient: agentClient}
}

// Report is called by log-agent (public; uses agent token inside payload).
func (h *AgentDiscoveryHandler) Report(c *gin.Context) {
	handleJSON(c, func(ctx context.Context, req service.AgentDiscoveryReportRequest) (*service.AgentDiscoveryReportResult, error) {
		items := make([]*pb.AgentDiscoveryItem, 0, len(req.Items))
		for _, it := range req.Items {
			extra := map[string]string{}
			for k, v := range it.Extra {
				if s, ok := v.(string); ok {
					extra[k] = s
				}
			}
			items = append(items, &pb.AgentDiscoveryItem{
				Kind:  it.Kind,
				Value: it.Value,
				Extra: extra,
			})
		}
		out, err := h.agentClient.ReportDiscovery(ctx, &pb.ReportDiscoveryRequest{
			Token: req.Token,
			Items: items,
		})
		if err != nil {
			return nil, grpcToAppError(err)
		}
		return &service.AgentDiscoveryReportResult{Accepted: int(out.GetAccepted())}, nil
	})
}

// List is used by UI (authz) to fetch discovery items for a project/server.
func (h *AgentDiscoveryHandler) List(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var q service.AgentDiscoveryListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	q.ProjectID = projectID
	req := &pb.ListDiscoveryRequest{
		ProjectId: uint64(q.ProjectID),
		ServerId:  uint64(q.ServerID),
		Limit:     int32(q.Limit),
	}
	if q.Kind != nil {
		req.Kind = *q.Kind
	}
	listResp, err := h.agentClient.ListDiscovery(c.Request.Context(), req)
	if err != nil {
		response.Error(c, grpcToAppError(err))
		return
	}
	list := make([]service.AgentDiscoveryListItem, 0, len(listResp.GetList()))
	for _, it := range listResp.GetList() {
		list = append(list, service.AgentDiscoveryListItem{
			Kind:       it.GetKind(),
			Value:      it.GetValue(),
			LastSeenAt: it.GetLastSeenAt(),
		})
	}
	response.Success(c, gin.H{"list": list})
}
