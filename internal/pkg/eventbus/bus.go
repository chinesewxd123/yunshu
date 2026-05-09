// Package eventbus 提供进程内、非阻塞事件分发（对齐 k8m eventbus：慢消费者丢事件，不阻塞发布方）。
// 不含 AI/MCP；供集群连接、Watch 等扩展点使用。
package eventbus

import (
	"log/slog"
	"sync"
)

// Type 事件类型。
type Type string

const (
	ClusterKomRegisterOK      Type = "cluster.kom.register.ok"
	ClusterKomRegisterFail    Type = "cluster.kom.register.fail"
	K8sResourceWatchStarted   Type = "k8s.watch.started"
	K8sResourceWatchClientClose Type = "k8s.watch.client_close"
)

// Event 负载由发布方约定。
type Event struct {
	Type    Type           `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

type Bus struct {
	mu           sync.RWMutex
	subscribers  map[Type][]chan Event
	defaultBufsz int
}

var (
	defaultBus = &Bus{
		subscribers:  make(map[Type][]chan Event),
		defaultBufsz: 1,
	}
)

// Default 返回进程内默认总线。
func Default() *Bus { return defaultBus }

// Subscribe 订阅；channel 缓冲为 1，满则丢事件。
func (b *Bus) Subscribe(t Type) <-chan Event {
	ch := make(chan Event, b.defaultBufsz)
	b.mu.Lock()
	b.subscribers[t] = append(b.subscribers[t], ch)
	b.mu.Unlock()
	return ch
}

// Publish 非阻塞发布。
func (b *Bus) Publish(e Event) {
	if e.Type == "" {
		return
	}
	b.mu.RLock()
	subs := append([]chan Event(nil), b.subscribers[e.Type]...)
	b.mu.RUnlock()

	dropped := 0
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		slog.Debug("eventbus dropped for slow consumers", "type", e.Type, "dropped", dropped)
	}
}
