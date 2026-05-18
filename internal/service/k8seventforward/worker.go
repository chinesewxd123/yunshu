package k8seventforward

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
)

type Worker struct {
	store    *Store
	client   *WebhookClient
	cfg      RuntimeConfig
	log      *slog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	interval time.Duration
	batch    int
	maxRetry int
	onBeforeBatch func()
	isEnabled     func() bool
}

func NewWorker(store *Store, client *WebhookClient, cfg RuntimeConfig, log *slog.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		store:    store,
		client:   client,
		cfg:      cfg,
		log:      log,
		ctx:      ctx,
		cancel:   cancel,
		interval: time.Duration(cfg.WorkerIntervalSeconds) * time.Second,
		batch:    cfg.WorkerBatchSize,
		maxRetry: cfg.WorkerMaxRetries,
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	go w.loop()
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) RefreshSettings(cfg RuntimeConfig) {
	w.cfg = cfg
	if cfg.WorkerIntervalSeconds > 0 {
		w.interval = time.Duration(cfg.WorkerIntervalSeconds) * time.Second
	}
	if cfg.WorkerBatchSize > 0 {
		w.batch = cfg.WorkerBatchSize
	}
	if cfg.WorkerMaxRetries > 0 {
		w.maxRetry = cfg.WorkerMaxRetries
	}
}

func (w *Worker) loop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(); err != nil {
				w.log.Warn("k8s event forward: batch failed", slog.Any("error", err))
			}
		}
	}
}

func (w *Worker) processBatch() error {
	if w.onBeforeBatch != nil {
		w.onBeforeBatch()
	}
	if w.isEnabled != nil && !w.isEnabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(w.ctx, 2*time.Minute)
	defer cancel()

	rules, err := w.store.ListEnabledRules(ctx)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	events, err := w.store.ListUnprocessed(ctx, w.batch)
	if err != nil || len(events) == 0 {
		return err
	}

	processedIDs := make(map[int64]bool)
	matchedIDs := make(map[int64]bool)

	for _, rule := range rules {
		clusters := ParseClusterIDSet(rule.ClusterIDs)
		filter := ParseRuleFilter(rule)
		webhookURL := w.resolveWebhookURL(rule.WebhookURL, w.cfg.UseInternalAlertWebhook)

		grouped := make(map[string][]model.K8sForwardedEvent)
		for i := range events {
			ev := events[i]
			if processedIDs[ev.ID] {
				continue
			}
			if ev.Attempts >= w.maxRetry {
				_ = w.store.MarkProcessed(ctx, ev.ID, true)
				processedIDs[ev.ID] = true
				continue
			}
			if _, ok := clusters[ev.ClusterID]; !ok {
				continue
			}
			if !filter.Match(&ev) {
				continue
			}
			matchedIDs[ev.ID] = true
			grouped[ev.ClusterID] = append(grouped[ev.ClusterID], ev)
		}

		for clusterID, batch := range grouped {
			if err := w.push(ctx, webhookURL, rule.Name, clusterID, batch); err != nil {
				w.log.Warn("k8s event forward: webhook push failed",
					slog.String("rule", rule.Name),
					slog.String("cluster_id", clusterID),
					slog.Any("error", err))
				for _, e := range batch {
					_ = w.store.IncrementAttempts(ctx, e.ID)
				}
				continue
			}
			for _, e := range batch {
				_ = w.store.MarkProcessed(ctx, e.ID, true)
				processedIDs[e.ID] = true
			}
		}
	}

	for _, ev := range events {
		if !processedIDs[ev.ID] && !matchedIDs[ev.ID] {
			_ = w.store.MarkProcessed(ctx, ev.ID, true)
		}
	}
	return nil
}

func (w *Worker) resolveWebhookURL(ruleURL string, useInternal bool) string {
	u := strings.TrimSpace(ruleURL)
	if useInternal && (u == "" || strings.EqualFold(u, "internal") || strings.EqualFold(u, "alertmanager")) {
		return w.cfg.AlertWebhookURL
	}
	return u
}

func (w *Worker) push(ctx context.Context, url, ruleName, clusterID string, events []model.K8sForwardedEvent) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("no webhook url for rule %s", ruleName)
	}
	var cid uint
	if id, err := strconv.ParseUint(clusterID, 10, 64); err == nil {
		cid = uint(id)
	}
	clusterName := w.store.GetClusterName(ctx, cid)
	payload := buildAlertManagerPayload(ruleName, clusterID, clusterName, events)
	return w.client.PostAlertmanager(ctx, url, payload)
}
