package handler

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"yunshu/internal/pkg/response"
	"yunshu/internal/pkg/sshclient"

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

	sess := newWSSession(c.Request.Context(), httpLog("http.ws.terminal"))
	defer sess.Cancel()
	defer sess.Wait()

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

	sess.Go("ping", func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sess.Context().Done():
				return
			case <-ticker.C:
				writeMu.Lock()
				err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
				writeMu.Unlock()
				if err != nil {
					sess.Cancel()
					return
				}
			}
		}
	})

	sess.Go("read", func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				sess.Cancel()
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
				sess.Cancel()
				_ = stdinW.Close()
				return
			default:
			}
		}
	})

	writeJSON(wsExecMessage{Type: "ready"})

	if err := h.svc.StreamServerTerminal(sess.Context(), projectID, serverID, stdinR, wsWriter, wsWriter, sizeCh); err != nil {
		writeJSON(wsExecMessage{Type: "error", Data: err.Error()})
		sess.Cancel()
		sess.Wait()
		return
	}
	writeJSON(wsExecMessage{Type: "exit"})
	sess.Cancel()
	sess.Wait()
}
