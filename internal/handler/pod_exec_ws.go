package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/pkg/response"

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
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		oh, err := url.Parse(origin)
		if err != nil || oh.Host == "" {
			return false
		}
		reqHost := strings.TrimSpace(r.Host)
		if reqHost == "" {
			return false
		}
		// 允许同源；开发环境 localhost 不同端口也放行。
		if strings.EqualFold(oh.Host, reqHost) {
			return true
		}
		ohName := strings.Split(oh.Host, ":")[0]
		reqName := strings.Split(reqHost, ":")[0]
		if ohName == reqName && (ohName == "localhost" || ohName == "127.0.0.1") {
			return true
		}
		return false
	},
}

func (h *PodHandler) ExecWS(c *gin.Context) {
	clusterID64, err := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	if err != nil || clusterID64 == 0 {
		response.Error(c, constants.ErrBadRequestWithMsg(constants.ErrMsgba2a155d1253))
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

	sess := newWSSession(c.Request.Context(), nil)
	defer sess.Cancel()
	defer sess.Wait()

	stdinR, stdinW := io.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	sizeCh := make(chan remotecommand.TerminalSize, 10)
	defer close(sizeCh)

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

	wsWriter := &wsTextWriter{write: func(p []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		err := conn.WriteJSON(wsExecMessage{Type: "stdout", Data: string(p)})
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}}

	conn.SetReadLimit(32 * 1024)
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
					case sizeCh <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
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
	writeText("\r\n")

	err = h.svc.ExecTTYStream(
		sess.Context(),
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
	sess.Cancel()
	sess.Wait()
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
