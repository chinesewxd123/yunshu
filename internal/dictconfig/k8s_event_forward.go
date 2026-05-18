package dictconfig

import (
	"context"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// K8sEventForwardDictTypes 数据字典中覆盖 k8s_event_forward.* 的 dict_type。
type K8sEventForwardDictTypes struct {
	Enabled               string
	WatcherBufferSize     string
	WorkerIntervalSeconds string
	WorkerBatchSize       string
	WorkerMaxRetries      string
}

func DefaultK8sEventForwardDictTypes() K8sEventForwardDictTypes {
	return K8sEventForwardDictTypes{
		Enabled:               "k8s_event_forward_enabled",
		WatcherBufferSize:     "k8s_event_forward_watcher_buffer_size",
		WorkerIntervalSeconds: "k8s_event_forward_worker_interval_seconds",
		WorkerBatchSize:       "k8s_event_forward_worker_batch_size",
		WorkerMaxRetries:      "k8s_event_forward_worker_max_retries",
	}
}

// ResolveK8sEventForwardConfig 以 yamlBase 为底，用已启用的数据字典项覆盖（字典存在则优先）。
func ResolveK8sEventForwardConfig(ctx context.Context, db *gorm.DB, yamlBase config.K8sEventForwardConfig, types K8sEventForwardDictTypes) config.K8sEventForwardConfig {
	cfg := yamlBase
	cfg.ApplyDefaults()
	if db == nil {
		return cfg
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.Enabled); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.Enabled = bv
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.WatcherBufferSize); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.WatcherBufferSize = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.WorkerIntervalSeconds); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.WorkerIntervalSeconds = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.WorkerBatchSize); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.WorkerBatchSize = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.WorkerMaxRetries); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.WorkerMaxRetries = n
		}
	}
	return cfg
}
