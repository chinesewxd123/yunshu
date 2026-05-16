package handler

import (
	"context"

	pb "yunshu/internal/grpc/proto"
	"yunshu/internal/pkg/constants"
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
	ServeJSON(c, func(ctx context.Context, req service.AgentDiscoveryReportRequest) (*service.AgentDiscoveryReportResult, error) {
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
		response.Error(c, constants.ErrBadRequestWithMsg(err.Error()))
		return
	}
	q.ProjectID = projectID
	list, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}
