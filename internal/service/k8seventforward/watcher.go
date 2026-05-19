package k8seventforward

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/model"
	"yunshu/internal/service"

	"github.com/robfig/cron/v3"
	"github.com/weibaohui/kom/kom"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	store    *Store
	runtime  *service.K8sRuntimeService
	cfg      RuntimeConfig
	log      *logx.Component
	eventCh  chan *model.K8sForwardedEvent
	ctx      context.Context
	cancel   context.CancelFunc
	activeMu sync.Mutex
	active   map[string]bool // clusterID -> watching
}

func NewWatcher(store *Store, runtime *service.K8sRuntimeService, cfg RuntimeConfig, log *logx.Component) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	buf := cfg.WatcherBufferSize
	if buf <= 0 {
		buf = 1000
	}
	return &Watcher{
		store:   store,
		runtime: runtime,
		cfg:     cfg,
		log:     log,
		eventCh: make(chan *model.K8sForwardedEvent, buf),
		ctx:     ctx,
		cancel:  cancel,
		active:  make(map[string]bool),
	}
}

func (w *Watcher) Start() {
	go w.persistLoop()
	go w.scheduleLoop()
}

func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Watcher) scheduleLoop() {
	inst := cron.New()
	_, _ = inst.AddFunc("@every 1m", func() {
		w.ensureWatches()
	})
	inst.Start()
	<-w.ctx.Done()
	inst.Stop()
}

func (w *Watcher) ensureWatches() {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	ids, err := w.store.ListEnabledClusterIDs(ctx)
	if err != nil {
		w.log.Warn("list clusters failed", slog.Any("error", err))
		return
	}
	for _, id := range ids {
		cid := strconv.FormatUint(uint64(id), 10)
		w.activeMu.Lock()
		if w.active[cid] {
			w.activeMu.Unlock()
			continue
		}
		w.active[cid] = true
		w.activeMu.Unlock()
		go w.watchCluster(cid, id)
	}
}

func (w *Watcher) watchCluster(clusterID string, id uint) {
	defer func() {
		w.activeMu.Lock()
		delete(w.active, clusterID)
		w.activeMu.Unlock()
	}()

	ctx := w.ctx
	if err := w.runtime.EnsureClusterRegistered(ctx, id); err != nil {
		w.log.Warn("register cluster failed",
			slog.String("cluster_id", clusterID), slog.Any("error", err))
		return
	}

	var watcher watch.Interface
	var evt eventsv1.Event
	if err := kom.Cluster(clusterID).WithContext(ctx).Resource(&evt).AllNamespace().Watch(&watcher).Error; err != nil {
		w.log.Warn("watch failed",
			slog.String("cluster_id", clusterID), slog.Any("error", err))
		return
	}
	defer watcher.Stop()

	w.log.Info("watching events", slog.String("cluster_id", clusterID))
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return
			}
			var typed eventsv1.Event
			if err := kom.Cluster(clusterID).WithContext(ctx).Tools().ConvertRuntimeObjectToTypedObject(e.Object, &typed); err != nil {
				w.log.Warn("convert event failed",
					slog.String("cluster_id", clusterID), slog.Any("error", err))
				continue
			}
			m := w.fromK8sEvent(clusterID, &typed)
			if m == nil || !m.ShouldForward() {
				continue
			}
			if err := w.enqueue(m); err != nil {
				w.log.Warn("enqueue failed",
					slog.String("evt_key", m.EvtKey), slog.Any("error", err))
			}
		}
	}
}

func (w *Watcher) fromK8sEvent(clusterID string, evt *eventsv1.Event) *model.K8sForwardedEvent {
	ts := time.Now()
	if !evt.EventTime.IsZero() {
		ts = evt.EventTime.Time
	} else if !evt.CreationTimestamp.IsZero() {
		ts = evt.CreationTimestamp.Time
	}
	key := string(evt.UID)
	if key == "" {
		key = fmt.Sprintf("%s/%s/%s/%s/%d", clusterID, evt.Regarding.Namespace, evt.Regarding.Name, evt.Reason, ts.UnixNano())
	}
	return &model.K8sForwardedEvent{
		EvtKey:    key,
		ClusterID: clusterID,
		Namespace: evt.Regarding.Namespace,
		Name:      evt.Regarding.Name,
		Type:      evt.Type,
		Reason:    evt.Reason,
		Level:     evt.Type,
		Message:   evt.Note,
		Timestamp: ts,
		Processed: false,
	}
}

func (w *Watcher) enqueue(ev *model.K8sForwardedEvent) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return fmt.Errorf("watcher stopped")
		case w.eventCh <- ev:
			return nil
		default:
			select {
			case <-timer.C:
			case <-w.ctx.Done():
				return fmt.Errorf("watcher stopped")
			}
		}
	}
}

func (w *Watcher) persistLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case ev, ok := <-w.eventCh:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := w.store.SaveEvent(ctx, ev)
			cancel()
			if err != nil {
				w.log.Warn("save event failed", slog.Any("error", err))
			}
		}
	}
}
