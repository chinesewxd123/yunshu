package service

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AlertMetrics AlertService 的 Prometheus 自监控指标
type AlertMetrics struct {
	// 规则评估
	RulesEvalTotal          *prometheus.CounterVec   // 规则评估总次数
	RulesEvalDuration       *prometheus.HistogramVec   // 规则评估耗时
	RulesEvalFailed         *prometheus.CounterVec     // 规则评估失败次数
	RulesPendingDuration    *prometheus.HistogramVec   // 告警pending到触发的时间

	// 告警接收
	AlertsReceivedTotal     prometheus.Counter       // 接收告警总数
	AlertsDroppedTotal      prometheus.Counter       // 丢弃告警总数

	// 聚合与抑制
	AggregateSuppressedTotal prometheus.Counter      // 聚合抑制总数
	FingerprintDedupTotal    prometheus.Counter      // 指纹去重抑制总数
	SilenceSuppressedTotal   prometheus.Counter      // 静默抑制总数
	InhibitionSuppressedTotal prometheus.Counter     // 告警抑制抑制总数

	// 通知发送
	NotificationsTotal      *prometheus.CounterVec   // 通知发送总次数
	NotificationsFailed     *prometheus.CounterVec   // 通知失败次数
	NotificationsDuration   *prometheus.HistogramVec // 通知耗时
	NotificationsQueued     prometheus.Gauge         // 队列中等待的通知数

	// 通道状态
	ChannelNotifications    *prometheus.CounterVec   // 按通道统计通知数

	// 队列
	EnrichQueueSize         prometheus.Gauge         // Prometheus增强队列大小
	EnrichQueueProcessed    prometheus.Counter       // 队列处理数
	EnrichQueueDropped      prometheus.Counter       // 队列丢弃数

	// 缓存
	// 抑制
	InhibitionRulesActive   prometheus.Gauge         // 活跃的抑制规则数
	InhibitionEventsTotal   prometheus.Counter       // 抑制事件总数

	// 数据源
	DatasourceQueryTotal    *prometheus.CounterVec   // 数据源查询总数
	DatasourceQueryFailed   *prometheus.CounterVec   // 数据源查询失败数
	DatasourceQueryDuration *prometheus.HistogramVec // 数据源查询耗时
}

// NewAlertMetrics 创建告警监控指标
func NewAlertMetrics() *AlertMetrics {
	return &AlertMetrics{
		// 规则评估指标
		RulesEvalTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "rules_eval_total",
			Help:      "Total number of alert rule evaluations",
		}, []string{"rule_id", "datasource_id"}),

		RulesEvalDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "rules_eval_duration_seconds",
			Help:      "Alert rule evaluation duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"rule_id", "datasource_id"}),

		RulesEvalFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "rules_eval_failed_total",
			Help:      "Total number of failed alert rule evaluations",
		}, []string{"rule_id", "datasource_id", "reason"}),

		RulesPendingDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "rules_pending_duration_seconds",
			Help:      "Time from pending to firing state",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		}, []string{"rule_id"}),

		// 告警接收指标
		AlertsReceivedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "alerts_received_total",
			Help:      "Total number of alerts received from alertmanager",
		}),

		AlertsDroppedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "alerts_dropped_total",
			Help:      "Total number of alerts dropped due to errors",
		}),

		// 聚合与抑制指标
		AggregateSuppressedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "aggregate_suppressed_total",
			Help:      "Total number of alerts suppressed by aggregation",
		}),

		FingerprintDedupTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "fingerprint_dedup_suppressed_total",
			Help:      "Total number of alerts suppressed by fingerprint deduplication",
		}),

		SilenceSuppressedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "silence_suppressed_total",
			Help:      "Total number of alerts suppressed by silence rules",
		}),

		InhibitionSuppressedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "inhibition_suppressed_total",
			Help:      "Total number of alerts suppressed by inhibition rules",
		}),

		// 通知发送指标
		NotificationsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "notifications_total",
			Help:      "Total number of notifications sent",
		}, []string{"channel_type", "status"}),

		NotificationsFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "notifications_failed_total",
			Help:      "Total number of failed notifications",
		}, []string{"channel_type", "reason"}),

		NotificationsDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "notifications_duration_seconds",
			Help:      "Notification sending duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"channel_type"}),

		NotificationsQueued: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "notifications_queued",
			Help:      "Number of notifications currently queued",
		}),

		ChannelNotifications: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "channel_notifications_total",
			Help:      "Total notifications by channel",
		}, []string{"channel_id", "channel_name", "channel_type", "status"}),

		// 队列指标
		EnrichQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "enrich_queue_size",
			Help:      "Current size of Prometheus enrich queue",
		}),

		EnrichQueueProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "enrich_queue_processed_total",
			Help:      "Total number of enrich tasks processed",
		}),

		EnrichQueueDropped: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "enrich_queue_dropped_total",
			Help:      "Total number of enrich tasks dropped due to full queue",
		}),

		// 缓存指标
		// 抑制指标
		InhibitionRulesActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "inhibition_rules_active",
			Help:      "Number of active inhibition rules",
		}),

		InhibitionEventsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "inhibition_events_total",
			Help:      "Total number of inhibition events",
		}),

		// 数据源指标
		DatasourceQueryTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "datasource_query_total",
			Help:      "Total number of datasource queries",
		}, []string{"datasource_id", "datasource_type", "query_type"}),

		DatasourceQueryFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "datasource_query_failed_total",
			Help:      "Total number of failed datasource queries",
		}, []string{"datasource_id", "datasource_type", "query_type", "reason"}),

		DatasourceQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "yunshu",
			Subsystem: "alert",
			Name:      "datasource_query_duration_seconds",
			Help:      "Datasource query duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
		}, []string{"datasource_id", "datasource_type", "query_type"}),
	}
}

// RecordRuleEval 记录规则评估
func (m *AlertMetrics) RecordRuleEval(ruleID, datasourceID uint, duration time.Duration, failed bool, reason string) {
	idStr := strconv.FormatUint(uint64(ruleID), 10)
	dsStr := strconv.FormatUint(uint64(datasourceID), 10)

	m.RulesEvalTotal.WithLabelValues(idStr, dsStr).Inc()
	m.RulesEvalDuration.WithLabelValues(idStr, dsStr).Observe(duration.Seconds())

	if failed {
		m.RulesEvalFailed.WithLabelValues(idStr, dsStr, reason).Inc()
	}
}

// RecordRulePending 记录pending状态
func (m *AlertMetrics) RecordRulePending(ruleID uint, pendingSeconds float64) {
	m.RulesPendingDuration.WithLabelValues(
		strconv.FormatUint(uint64(ruleID), 10),
	).Observe(pendingSeconds)
}

// RecordAlertReceived 记录告警接收
func (m *AlertMetrics) RecordAlertReceived(count int) {
	m.AlertsReceivedTotal.Add(float64(count))
}

// RecordAlertDropped 记录告警丢弃
func (m *AlertMetrics) RecordAlertDropped(count int) {
	m.AlertsDroppedTotal.Add(float64(count))
}

// RecordAggregateSuppressed 记录聚合抑制
func (m *AlertMetrics) RecordAggregateSuppressed() {
	m.AggregateSuppressedTotal.Inc()
}

// RecordFingerprintDedup 记录指纹去重
func (m *AlertMetrics) RecordFingerprintDedup() {
	m.FingerprintDedupTotal.Inc()
}

// RecordSilenceSuppressed 记录静默抑制
func (m *AlertMetrics) RecordSilenceSuppressed() {
	m.SilenceSuppressedTotal.Inc()
}

// RecordInhibitionSuppressed 记录告警抑制
func (m *AlertMetrics) RecordInhibitionSuppressed() {
	m.InhibitionSuppressedTotal.Inc()
}

// RecordNotification 记录通知发送
func (m *AlertMetrics) RecordNotification(channelType string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.NotificationsTotal.WithLabelValues(channelType, status).Inc()
	m.NotificationsDuration.WithLabelValues(channelType).Observe(duration.Seconds())
}

// RecordNotificationFailed 记录通知失败
func (m *AlertMetrics) RecordNotificationFailed(channelType, reason string) {
	m.NotificationsFailed.WithLabelValues(channelType, reason).Inc()
}

// RecordChannelNotification 记录按通道统计
func (m *AlertMetrics) RecordChannelNotification(channelID uint, channelName, channelType string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.ChannelNotifications.WithLabelValues(
		strconv.FormatUint(uint64(channelID), 10),
		channelName,
		channelType,
		status,
	).Inc()
}

// RecordDatasourceQuery 记录数据源查询
func (m *AlertMetrics) RecordDatasourceQuery(datasourceID uint, datasourceType, queryType string, duration time.Duration, failed bool, reason string) {
	idStr := strconv.FormatUint(uint64(datasourceID), 10)
	m.DatasourceQueryTotal.WithLabelValues(idStr, datasourceType, queryType).Inc()
	m.DatasourceQueryDuration.WithLabelValues(idStr, datasourceType, queryType).Observe(duration.Seconds())

	if failed {
		m.DatasourceQueryFailed.WithLabelValues(idStr, datasourceType, queryType, reason).Inc()
	}
}

// SetEnrichQueueSize 设置增强队列大小
func (m *AlertMetrics) SetEnrichQueueSize(size float64) {
	m.EnrichQueueSize.Set(size)
}

// RecordEnrichProcessed 记录队列处理
func (m *AlertMetrics) RecordEnrichProcessed() {
	m.EnrichQueueProcessed.Inc()
}

// RecordEnrichDropped 记录队列丢弃
func (m *AlertMetrics) RecordEnrichDropped() {
	m.EnrichQueueDropped.Inc()
}

// SetInhibitionRulesActive 设置活跃抑制规则数
func (m *AlertMetrics) SetInhibitionRulesActive(count float64) {
	m.InhibitionRulesActive.Set(count)
}

// RecordInhibitionEvent 记录抑制事件
func (m *AlertMetrics) RecordInhibitionEvent() {
	m.InhibitionEventsTotal.Inc()
}

// AlertMetricsUpdater 用于定期更新指标
type AlertMetricsUpdater struct {
	metrics   *AlertMetrics
	inhibitionSvc *AlertInhibitionService
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewAlertMetricsUpdater 创建指标更新器
func NewAlertMetricsUpdater(metrics *AlertMetrics, inhibitionSvc *AlertInhibitionService) *AlertMetricsUpdater {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertMetricsUpdater{
		metrics:       metrics,
		inhibitionSvc: inhibitionSvc,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动定期更新
func (u *AlertMetricsUpdater) Start() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-u.ctx.Done():
				return
			case <-ticker.C:
				u.update()
			}
		}
	}()
}

// Stop 停止更新器
func (u *AlertMetricsUpdater) Stop() {
	u.cancel()
}

func (u *AlertMetricsUpdater) update() {
	// 更新活跃抑制规则数
	if u.inhibitionSvc != nil {
		rules, _ := u.inhibitionSvc.ListEnabledRules(u.ctx)
		u.metrics.SetInhibitionRulesActive(float64(len(rules)))
	}
}
