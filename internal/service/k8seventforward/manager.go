package k8seventforward

import (
	"context"
	"fmt"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/model"
	"yunshu/internal/service"
	"yunshu/internal/service/svclog"

	"gorm.io/gorm"
)

// RuntimeConfig 运行期合并字典/YAML 与 DB 规则表全局参数。
type RuntimeConfig struct {
	WatcherBufferSize     int
	WorkerIntervalSeconds int
	WorkerBatchSize       int
	WorkerMaxRetries      int
	// AlertWebhookURL 本机告警平台入站地址（复用 /alerts/webhook/alertmanager）
	AlertWebhookURL         string
	UseInternalAlertWebhook bool
}

// Manager 协调 Watcher 与 Worker（参考 k8m eventhandler）。
type Manager struct {
	store    *Store
	watcher  *Watcher
	worker   *Worker
	log      *logx.Component
	enabled  bool
	db       *gorm.DB
	yamlBase config.K8sEventForwardConfig
	appPort  int
}

func NewManager(
	db *gorm.DB,
	runtime *service.K8sRuntimeService,
	yamlBase config.K8sEventForwardConfig,
	alertCfg config.AlertConfig,
	appPort int,
	log *logx.Component,
) (*Manager, error) {
	if log == nil {
		log = svclog.Worker("k8s.event_forward")
	}
	ctx := context.Background()
	resolved := dictconfig.ResolveK8sEventForwardConfig(ctx, db, yamlBase, dictconfig.DefaultK8sEventForwardDictTypes())

	store := NewStore(db)
	defaults := model.K8sEventForwardSetting{
		ID:                     1,
		ProcessIntervalSeconds: resolved.WorkerIntervalSeconds,
		BatchSize:              resolved.WorkerBatchSize,
		MaxRetries:             resolved.WorkerMaxRetries,
		WatcherBufferSize:      resolved.WatcherBufferSize,
	}
	if err := store.EnsureDefaultSettings(ctx, defaults); err != nil {
		return nil, err
	}

	rt, err := loadRuntimeConfig(store, resolved, appPort)
	if err != nil {
		return nil, err
	}

	client := NewWebhookClient(alertCfg.WebhookToken, 0)
	mgr := &Manager{
		store:    store,
		watcher:  NewWatcher(store, runtime, rt, log),
		worker:   NewWorker(store, client, rt, log),
		log:      log,
		enabled:  resolved.Enabled,
		db:       db,
		yamlBase: yamlBase,
		appPort:  appPort,
	}
	mgr.worker.onBeforeBatch = mgr.reloadRuntimeConfig
	mgr.worker.isEnabled = func() bool { return mgr.enabled }
	return mgr, nil
}

func (m *Manager) reloadRuntimeConfig() {
	if m == nil || m.db == nil {
		return
	}
	ctx := context.Background()
	resolved := dictconfig.ResolveK8sEventForwardConfig(ctx, m.db, m.yamlBase, dictconfig.DefaultK8sEventForwardDictTypes())
	m.enabled = resolved.Enabled
	rt, err := loadRuntimeConfig(m.store, resolved, m.appPort)
	if err != nil {
		m.log.Warnw("Failed to reload K8s event forward config", "error", err)
		return
	}
	m.worker.RefreshSettings(rt)
}

func loadRuntimeConfig(store *Store, appCfg config.K8sEventForwardConfig, port int) (RuntimeConfig, error) {
	st, err := store.LoadSettings(context.Background())
	if err != nil {
		return RuntimeConfig{}, err
	}
	rt := RuntimeConfig{
		WatcherBufferSize:       firstPositive(st.WatcherBufferSize, appCfg.WatcherBufferSize),
		WorkerIntervalSeconds:   firstPositive(st.ProcessIntervalSeconds, appCfg.WorkerIntervalSeconds),
		WorkerBatchSize:         firstPositive(st.BatchSize, appCfg.WorkerBatchSize),
		WorkerMaxRetries:        firstPositive(st.MaxRetries, appCfg.WorkerMaxRetries),
		UseInternalAlertWebhook: true,
	}
	if port <= 0 {
		port = 8080
	}
	rt.AlertWebhookURL = fmt.Sprintf("http://127.0.0.1:%d/api/v1/alerts/webhook/alertmanager", port)
	return rt, nil
}

func firstPositive(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

func (m *Manager) Start() {
	if !m.enabled {
		m.log.Infow("K8s event forward disabled in config")
		return
	}
	ctx := context.Background()
	ok, err := m.store.HasEnabledRules(ctx)
	if err != nil {
		m.log.Warnw("Failed to check K8s event forward rules", "error", err)
		return
	}
	if !ok {
		m.log.Infow("No enabled K8s event forward rules, watcher and worker not started")
		return
	}
	m.log.Infow("Starting K8s event forward watcher and worker")
	m.watcher.Start()
	m.worker.Start()
}

func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
	if m.worker != nil {
		m.worker.Stop()
	}
}
