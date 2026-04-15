package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type LogAgentHandler struct {
	svc *service.LogAgentService
}

func NewLogAgentHandler(svc *service.LogAgentService) *LogAgentHandler {
	return &LogAgentHandler{svc: svc}
}

func (h *LogAgentHandler) Register(c *gin.Context) {
	handleJSON(c, h.svc.Register)
}

func (h *LogAgentHandler) PublicRegister(c *gin.Context) {
	handleJSON(c, h.svc.PublicRegister)
}

func (h *LogAgentHandler) RuntimeConfig(c *gin.Context) {
	token := c.Query("token")
	data, err := h.svc.RuntimeConfigByToken(c.Request.Context(), token)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

type agentStatusQuery struct {
	ServerID    uint `form:"server_id" binding:"required"`
	LogSourceID uint `form:"log_source_id"`
}

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
	data, err := h.svc.Status(c.Request.Context(), projectID, q.ServerID, q.LogSourceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *LogAgentHandler) Bootstrap(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.AgentBootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	req.ProjectID = projectID
	data, err := h.svc.Bootstrap(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *LogAgentHandler) RotateToken(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.AgentBootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	req.ProjectID = projectID
	data, err := h.svc.RotateToken(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

var agentWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type agentLogMessage struct {
	ProjectID   uint     `json:"project_id"`
	ServerID    uint     `json:"server_id"`
	LogSourceID uint     `json:"log_source_id"`
	Line        string   `json:"line"`
	Lines       []string `json:"lines"`
	FilePath    string   `json:"file_path,omitempty"`
	FilePaths   []string `json:"file_paths,omitempty"`
	Entries     []struct {
		Line     string `json:"line"`
		FilePath string `json:"file_path,omitempty"`
	} `json:"entries,omitempty"`
	Seq uint64 `json:"seq"`
}

func (h *LogAgentHandler) IngestWS(c *gin.Context) {
	token := c.Query("token")
	agent, err := h.svc.AuthenticateByToken(c.Request.Context(), token)
	if err != nil {
		response.Error(c, err)
		return
	}
	conn, err := agentWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	// Throttle DB heartbeat updates: at most once per minute.
	lastSeenTouch := time.Time{}
	touchSeen := func(force bool) {
		now := time.Now()
		if !force && !lastSeenTouch.IsZero() && now.Sub(lastSeenTouch) < time.Minute {
			return
		}
		h.svc.TouchSeen(c.Request.Context(), agent.ID)
		lastSeenTouch = now
	}
	// Mark online immediately after WS established.
	touchSeen(true)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		// Heartbeat: keep agent online even if log stream is quiet.
		touchSeen(false)
		return nil
	})
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(3*time.Second))
		}
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg agentLogMessage
		if e := json.Unmarshal(raw, &msg); e != nil {
			continue
		}
		if msg.ProjectID == 0 || msg.ServerID == 0 || msg.LogSourceID == 0 {
			continue
		}
		if msg.ProjectID != agent.ProjectID || msg.ServerID != agent.ServerID {
			continue
		}
		key := service.BuildLogStreamKey(msg.ProjectID, msg.ServerID, msg.LogSourceID)
		if msg.Line != "" {
			service.AgentLogBroker.Publish(key, service.AgentLogEvent{
				Line:     msg.Line,
				FilePath: msg.FilePath,
			})
		}
		for i, ln := range msg.Lines {
			if ln == "" {
				continue
			}
			fp := ""
			if i < len(msg.FilePaths) {
				fp = msg.FilePaths[i]
			}
			service.AgentLogBroker.Publish(key, service.AgentLogEvent{
				Line:     ln,
				FilePath: fp,
			})
		}
		for _, it := range msg.Entries {
			if it.Line == "" {
				continue
			}
			service.AgentLogBroker.Publish(key, service.AgentLogEvent{
				Line:     it.Line,
				FilePath: it.FilePath,
			})
		}
		// lightweight ACK for agent-side resend window management
		if msg.Seq > 0 {
			_ = conn.WriteJSON(gin.H{"type": "ack", "seq": msg.Seq, "ts": time.Now().UnixMilli()})
		}
		touchSeen(false)
	}
}
