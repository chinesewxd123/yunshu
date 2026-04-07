package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

type wsExecMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

type wsTerminalSizeQueue struct {
	ch <-chan remotecommand.TerminalSize
}

func (q *wsTerminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		// same-origin by default; allow if origin is empty (some clients)
		return true
	},
}

func (h *PodHandler) ExecWS(c *gin.Context) {
	clusterID64, err := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	if err != nil || clusterID64 == 0 {
		response.Error(c, apperror.BadRequest("cluster_id 非法"))
		return
	}
	namespace := c.Query("namespace")
	name := c.Query("name")
	container := c.Query("container")

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

	sizeCh := make(chan remotecommand.TerminalSize, 10)
	defer close(sizeCh)

	// ws write must be serialized
	var writeMu sync.Mutex
	writeJSON := func(msg wsExecMessage) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteJSON(msg)
	}
	writeText := func(s string) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteMessage(websocket.TextMessage, []byte(s))
	}

	// output writer -> websocket
	wsWriter := &wsTextWriter{write: func(p []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		err := conn.WriteJSON(wsExecMessage{Type: "stdout", Data: string(p)})
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}}

	// read loop: stdin + resize + ping
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
					case sizeCh <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
					default:
					}
				}
			case "close":
				cancel()
				_ = stdinW.Close()
				return
			default:
				// ignore
			}
		}
	}()

	writeJSON(wsExecMessage{Type: "ready"})
	writeText("\r\n")

	err = h.svc.ExecTTYStream(
		ctx,
		uint(clusterID64),
		namespace,
		name,
		container,
		stdinR,
		wsWriter,
		wsWriter,
		&wsTerminalSizeQueue{ch: sizeCh},
	)
	if err != nil {
		writeJSON(wsExecMessage{Type: "error", Data: err.Error()})
	} else {
		writeJSON(wsExecMessage{Type: "exit"})
	}
}

type wsTextWriter struct {
	write func([]byte) (int, error)
}

func (w *wsTextWriter) Write(p []byte) (int, error) {
	if w == nil || w.write == nil {
		return len(p), nil
	}
	return w.write(p)
}
