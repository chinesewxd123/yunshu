package handler

import (
	"context"
	"log/slog"
	"sync"

	logx "yunshu/internal/pkg/logger"
)

// wsSession 协调 WebSocket 辅助 goroutine（读循环、Ping 等），避免主流程返回后 goroutine 泄露。
type wsSession struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	log    *logx.Component
}

func newWSSession(parent context.Context, log *logx.Component) *wsSession {
	ctx, cancel := context.WithCancel(parent)
	return &wsSession{ctx: ctx, cancel: cancel, log: log}
}

func (s *wsSession) Context() context.Context {
	return s.ctx
}

func (s *wsSession) Cancel() {
	s.cancel()
}

// Go 启动带 WaitGroup 与 panic 恢复的 goroutine；panic 时会 cancel 会话上下文。
func (s *wsSession) Go(name string, fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				if s.log != nil {
					s.log.Error("websocket goroutine panic", slog.String("name", name), slog.Any("error", r))
				}
				s.cancel()
			}
		}()
		fn()
	}()
}

// Wait 阻塞直到所有通过 Go 启动的 goroutine 退出。
func (s *wsSession) Wait() {
	s.wg.Wait()
}
