package config

// K8sEventForwardConfig 多集群 K8s Event 监听与转发（参考 k8m eventhandler）。
// 运行期以数据字典优先、config.yaml 兜底；见 dictconfig.ResolveK8sEventForwardConfig。
type K8sEventForwardConfig struct {
	Enabled               bool `mapstructure:"enabled"`
	WatcherBufferSize     int  `mapstructure:"watcher_buffer_size"`
	WorkerIntervalSeconds int  `mapstructure:"worker_interval_seconds"`
	WorkerBatchSize       int  `mapstructure:"worker_batch_size"`
	WorkerMaxRetries      int  `mapstructure:"worker_max_retries"`
}

func (c *K8sEventForwardConfig) ApplyDefaults() {
	if c.WatcherBufferSize <= 0 {
		c.WatcherBufferSize = 1000
	}
	if c.WorkerIntervalSeconds <= 0 {
		c.WorkerIntervalSeconds = 10
	}
	if c.WorkerBatchSize <= 0 {
		c.WorkerBatchSize = 50
	}
	if c.WorkerMaxRetries <= 0 {
		c.WorkerMaxRetries = 3
	}
}
