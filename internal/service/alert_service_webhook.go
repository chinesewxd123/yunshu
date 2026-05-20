package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
)

func (s *AlertService) logSilenceSuppressed(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, silenceID uint, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          alertEventSourceFromPayload(payload),
		Title:           title + " (silence suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: monitorPipelineFromPayload(payload),
		GroupKey:        groupKey,
		LabelsDigest:    labelsDigest,
		ChannelID:       0,
		ChannelName:     "（静默抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    "silence_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by platform silence_id=%d", silenceID), s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) logNoMatchedChannel(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, payload map[string]interface{}, reason string) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             alertEventSourceFromPayload(payload),
		Title:              title + " (no matched channel)",
		Severity:           severity,
		Status:             status,
		Cluster:            cluster,
		MonitorPipeline:    monitorPipelineFromPayload(payload),
		GroupKey:           groupKey,
		LabelsDigest:       labelsDigest,
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matchedPolicyNames"),
		ChannelID:          0,
		ChannelName:        "（无匹配通道）",
		Success:            false,
		HTTPStatusCode:     0,
		ErrorMessage:       reason,
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload:    "",
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) logAllChannelsDeliveryFailed(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             alertEventSourceFromPayload(payload),
		Title:              title + " (all channel delivery failed)",
		Severity:           severity,
		Status:             status,
		Cluster:            cluster,
		MonitorPipeline:    monitorPipelineFromPayload(payload),
		GroupKey:           groupKey,
		LabelsDigest:       labelsDigest,
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matchedPolicyNames"),
		ChannelID:          0,
		ChannelName:        "（外发失败·全部通道）",
		Success:            false,
		HTTPStatusCode:     0,
		ErrorMessage:       "all_channel_delivery_failed",
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload:    "",
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

// ReceiveAlertmanager 执行对应的业务逻辑。
// 配置启用且 Redis 可用时，Webhook 先入队异步消费；内置评估路径应调用 receiveAlertmanagerPayloadSync 避免二次入队。
func (s *AlertService) ReceiveAlertmanager(ctx context.Context, payload AlertManagerPayload) error {
	if s.shouldEnqueueAlertmanagerWebhook() {
		if err := s.enqueueAlertmanagerWebhook(ctx, payload); err != nil {
			s.logWebhookWarn("Failed to enqueue alert webhook, processing synchronously",
				append(webhookPayloadLogAttrs(payload), "error", err)...)
			return s.receiveAlertmanagerPayloadSync(ctx, payload)
		}
		return nil
	}
	return s.receiveAlertmanagerPayloadSync(ctx, payload)
}

func alertEventSourceFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return "alertmanager"
	}
	src := strings.TrimSpace(fmt.Sprintf("%v", payload["source"]))
	if src == "" {
		return "alertmanager"
	}
	return src
}

// ValidateWebhookToken 校验相关的业务逻辑。
func (s *AlertService) ValidateWebhookToken(token string) bool {
	expected := strings.TrimSpace(s.cfg.WebhookToken)
	if expected == "" {
		return false
	}
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token) == expected
}

// resolveAlertDatasourceMeta 统一解析「数据源」维度：优先标签/规则上的 datasource_id，其次平台规则反查；monitor_pipeline 列写入简短 slug 便于筛选。
func (s *AlertService) resolveAlertDatasourceMeta(ctx context.Context, labels map[string]string, receiver string) (dsID uint, dsName, dsType, pipelineSlug string) {
	rcv := strings.TrimSpace(receiver)
	if rcv == "cloud-expiry" {
		return 0, "云资源到期", "cloud_expiry", "cloud_expiry"
	}
	for _, key := range []string{"yunshu_datasource_id", "datasource_id"} {
		if labels == nil {
			break
		}
		if n, ok := parseLabelUint(labels[key]); ok && n > 0 {
			dsID = n
			break
		}
	}
	if dsID == 0 && s.isPlatformMonitor(labels, receiver) {
		if rid, ok := parseLabelUint(labels["monitor_rule_id"]); ok && rid > 0 {
			var rule model.AlertMonitorRule
			if err := s.db.WithContext(ctx).First(&rule, rid).Error; err == nil && rule.DatasourceID > 0 {
				dsID = rule.DatasourceID
			}
		}
	}
	if dsID > 0 {
		var ds model.AlertDatasource
		if err := s.db.WithContext(ctx).First(&ds, dsID).Error; err == nil {
			dsName = strings.TrimSpace(ds.Name)
			dsType = strings.TrimSpace(ds.Type)
		}
		if dsType == "" {
			dsType = "prometheus"
		}
		return dsID, dsName, dsType, fmt.Sprintf("ds:%d", dsID)
	}
	if labels != nil {
		if n := strings.TrimSpace(labels["datasource_name"]); n != "" {
			dsName = n
		}
		if t := strings.TrimSpace(labels["datasource_type"]); t != "" {
			dsType = t
		}
	}
	if dsType == "" {
		dsType = "prometheus"
	}
	if s.isPlatformMonitor(labels, receiver) {
		if dsName == "" {
			dsName = "平台监控规则"
		}
		return 0, dsName, dsType, "platform_monitor"
	}
	return 0, dsName, dsType, "alertmanager"
}

func fillAlertEventDatasourceFromPayload(ev *model.AlertEvent, payload map[string]interface{}) {
	if ev == nil || payload == nil {
		return
	}
	if id := payloadUintAny(payload["datasourceId"]); id > 0 {
		ev.DatasourceID = id
	}
	if s := strings.TrimSpace(fmt.Sprintf("%v", payload["datasourceName"])); s != "" && s != "<nil>" {
		ev.DatasourceName = truncateText(s, 128)
	}
	if s := strings.TrimSpace(fmt.Sprintf("%v", payload["datasourceType"])); s != "" && s != "<nil>" {
		ev.DatasourceType = truncateText(s, 32)
	}
}

func payloadUintAny(v interface{}) uint {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case uint:
		return x
	case uint32:
		return uint(x)
	case uint64:
		if x <= 1<<32-1 {
			return uint(x)
		}
	case int:
		if x > 0 {
			return uint(x)
		}
	case int64:
		if x > 0 && x < 1<<32 {
			return uint(x)
		}
	case float64:
		if x > 0 && x < 1e12 {
			return uint(x)
		}
	case string:
		if n, ok := parseLabelUint(x); ok {
			return n
		}
	case json.Number:
		if u, err := x.Int64(); err == nil && u > 0 {
			return uint(u)
		}
	}
	return 0
}

func (s *AlertService) isPlatformMonitor(labels map[string]string, receiver string) bool {
	if strings.TrimSpace(receiver) == "platform-monitor" {
		return true
	}
	if labels != nil && strings.TrimSpace(labels["source"]) == "prometheus_monitor" {
		return true
	}
	return false
}

func (s *AlertService) resolveAlertEnvironmentLabel(labels map[string]string, receiver string, dims alertnotify.Dims, alertLabels map[string]string) string {
	if s.isPlatformMonitor(labels, receiver) {
		// 集群列仅承载「环境/cluster」语义；平台规则未同步 cluster 标签时不要写死 unknown，回退到实例/节点等展示维度。
		if v := strings.TrimSpace(labels["cluster"]); v != "" {
			return v
		}
		return alertnotify.InferEnvironmentDisplay(labels, dims)
	}
	envLabel := strings.TrimSpace(dims.Cluster)
	if envLabel == "" {
		envLabel = alertnotify.InferEnvironmentDisplay(alertLabels, dims)
	}
	return envLabel
}

func monitorPipelineFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"]))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func mergeStringMap(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// AlertDatasourceFilterOption 历史告警筛选：已出现过的数据源（按事件表聚合）。
type AlertDatasourceFilterOption struct {
	ID   uint   `json:"id" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`
}

type AlertHistoryStats struct {
	Total                   int64                         `json:"total"`
	Firing                  int64                         `json:"firing"`
	Resolved                int64                         `json:"resolved"`
	Success                 int64                         `json:"success"`
	Failed                  int64                         `json:"failed"`
	TodayCreated            int64                         `json:"today_created"`
	ClusterValues           []string                      `json:"cluster_values"`
	MonitorPipelineValues   []string                      `json:"monitor_pipeline_values"`
	DatasourceFilterOptions []AlertDatasourceFilterOption `json:"datasource_filter_options"`
}

func (s *AlertService) HistoryStats(ctx context.Context) (*AlertHistoryStats, error) {
	stats := &AlertHistoryStats{}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	var agg struct {
		Total        int64
		Firing       int64
		Resolved     int64
		Success      int64
		Failed       int64
		TodayCreated int64
	}
	// 单次聚合扫描，避免对 alert_events 连续 6 次 COUNT。
	if err := s.db.WithContext(ctx).Raw(`
SELECT
  COUNT(*) AS total,
  COALESCE(SUM(CASE WHEN status = 'firing' THEN 1 ELSE 0 END), 0) AS firing,
  COALESCE(SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END), 0) AS resolved,
  COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) AS success,
  COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) AS failed,
  COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS today_created
FROM alert_events
WHERE deleted_at IS NULL`, dayStart, dayEnd).Scan(&agg).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "HistoryStats", err)
	}
	stats.Total = agg.Total
	stats.Firing = agg.Firing
	stats.Resolved = agg.Resolved
	stats.Success = agg.Success
	stats.Failed = agg.Failed
	stats.TodayCreated = agg.TodayCreated
	var clusters []string
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(cluster, '')) != ''").
		Group("cluster").
		Order("cluster ASC").
		Limit(500).
		Pluck("cluster", &clusters).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "HistoryStats", err)
	}
	stats.ClusterValues = clusters
	var pipes []string
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(monitor_pipeline, '')) != ''").
		Group("monitor_pipeline").
		Order("monitor_pipeline ASC").
		Limit(32).
		Pluck("monitor_pipeline", &pipes).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "HistoryStats", err)
	}
	stats.MonitorPipelineValues = pipes
	var dsRows []AlertDatasourceFilterOption
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("datasource_id AS id, MAX(datasource_name) AS name").
		Where("datasource_id > ?", 0).
		Group("datasource_id").
		Order("id DESC").
		Limit(200).
		Scan(&dsRows).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "HistoryStats", err)
	}
	stats.DatasourceFilterOptions = dsRows
	return stats, nil
}

// applyAlertEventProjectFilter 按项目收窄历史告警：数据源归属或 request_payload 中的 project_id。
func applyAlertEventProjectFilter(tx *gorm.DB, db *gorm.DB, projectID uint) *gorm.DB {
	if projectID == 0 || tx == nil || db == nil {
		return tx
	}
	dsSub := db.Model(&model.AlertDatasource{}).Select("id").Where("project_id = ?", projectID)
	pid := fmt.Sprintf("%d", projectID)
	return tx.Where(
		"datasource_id IN (?) OR request_payload LIKE ? OR request_payload LIKE ?",
		dsSub,
		`%"project_id":"`+pid+`"%`,
		`%"project_id":`+pid+`,%`,
	)
}
