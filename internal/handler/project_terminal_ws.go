package handler

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"go-permission-system/internal/pkg/response"
	"go-permission-system/internal/pkg/sshclient"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (h *ProjectHandler) ServerTerminalWS(c *gin.Context) {
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

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	stdinR, stdinW := io.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	sizeCh := make(chan sshclient.TerminalSize, 10)
	defer close(sizeCh)

	var writeMu sync.Mutex
	writeJSON := func(msg wsExecMessage) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteJSON(msg)
	}

	wsWriter := &wsTextWriter{write: func(p []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := conn.WriteJSON(wsExecMessage{Type: "stdout", Data: string(p)}); err != nil {
			return 0, err
		}
		return len(p), nil
	}}

	conn.SetReadLimit(2 * 1024 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				writeMu.Lock()
				_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
				writeMu.Unlock()
			}
		}
	}()

	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				cancel()
				_ = stdinW.Close()
				return
			}
			var msg wsExecMessage
			if e := json.Unmarshal(raw, &msg); e != nil {
				continue
			}
			switch msg.Type {
			case "input":
				if msg.Data != "" {
					_, _ = stdinW.Write([]byte(msg.Data))
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					select {
					case sizeCh <- sshclient.TerminalSize{Cols: msg.Cols, Rows: msg.Rows}:
					default:
					}
				}
			case "close":
				cancel()
				_ = stdinW.Close()
				return
			default:
			}
		}
	}()

	writeJSON(wsExecMessage{Type: "ready"})

	if err := h.svc.StreamServerTerminal(ctx, projectID, serverID, stdinR, wsWriter, wsWriter, sizeCh); err != nil {
		writeJSON(wsExecMessage{Type: "error", Data: err.Error()})
		return
	}
	writeJSON(wsExecMessage{Type: "exit"})
}
