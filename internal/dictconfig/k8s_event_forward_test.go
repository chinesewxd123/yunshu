package dictconfig

import (
	"context"
	"testing"

	"yunshu/internal/config"
)

func TestResolveK8sEventForwardConfig_defaults(t *testing.T) {
	base := config.K8sEventForwardConfig{Enabled: false, WatcherBufferSize: 500}
	got := ResolveK8sEventForwardConfig(context.Background(), nil, base, DefaultK8sEventForwardDictTypes())
	if got.WatcherBufferSize != 500 {
		t.Fatalf("expected buffer 500, got %d", got.WatcherBufferSize)
	}
	if got.WorkerIntervalSeconds != 10 {
		t.Fatalf("expected default interval 10, got %d", got.WorkerIntervalSeconds)
	}
}
