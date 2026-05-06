package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
)

// RecordSourceInhibition 记录源告警并检查是否需要抑制
func (s *AlertService) RecordSourceInhibition(ctx context.Context, labels map[string]string) error {
	if s.inhibitionSvc == nil {
		return nil
	}

	// 检查是否匹配任何源告警规则
	ruleIDs, err := s.inhibitionSvc.CheckSourceMatch(ctx, labels)
	if err != nil || len(ruleIDs) == 0 {
		return err
	}

	// 提取指纹
	fp := labels["fingerprint"]
	if fp == "" {
		fp = s.computeFingerprintFromLabels(labels)
	}

	// 记录每个匹配的源规则
	for _, ruleID := range ruleIDs {
		if err := s.inhibitionSvc.RecordSourceAlert(ctx, ruleID, fp, labels); err != nil {
			continue
		}
	}

	return nil
}

// CheckInhibition 检查告警是否被抑制
func (s *AlertService) CheckInhibition(ctx context.Context, labels map[string]string) (bool, *model.AlertInhibitionEvent) {
	if s.inhibitionSvc == nil {
		return false, nil
	}

	inhibited, event, err := s.inhibitionSvc.CheckInhibition(ctx, labels)
	if err != nil {
		return false, nil
	}

	return inhibited, event
}

// ClearSourceInhibition 当告警恢复时清除源告警记录
func (s *AlertService) ClearSourceInhibition(ctx context.Context, labels map[string]string) error {
	if s.inhibitionSvc == nil {
		return nil
	}

	ruleIDs, err := s.inhibitionSvc.CheckSourceMatch(ctx, labels)
	if err != nil {
		return err
	}

	fp := labels["fingerprint"]
	if fp == "" {
		fp = s.computeFingerprintFromLabels(labels)
	}

	for _, ruleID := range ruleIDs {
		if err := s.inhibitionSvc.ClearSourceAlert(ctx, ruleID, fp); err != nil {
			continue
		}
	}

	return nil
}

// computeFingerprintFromLabels 从标签计算简易指纹
func (s *AlertService) computeFingerprintFromLabels(labels map[string]string) string {
	var sb strings.Builder
	keys := []string{"alertname", "cluster", "namespace", "instance", "pod", "node"}
	for _, k := range keys {
		if v := labels[k]; v != "" {
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			sb.WriteString(";")
		}
	}
	// 如果没有标准标签，使用所有标签
	if sb.Len() == 0 {
		for k, v := range labels {
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			sb.WriteString(";")
		}
	}

	return fmt.Sprintf("%x", sb.String())[:32]
}

// logInhibitionEvent 记录抑制事件到告警历史
func (s *AlertService) logInhibitionEvent(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, event *model.AlertInhibitionEvent, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	e := model.AlertEvent{
		Source:          "alertmanager",
		Title:           title + " (inhibition suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: monitorPipelineFromPayload(payload),
		GroupKey:        groupKey,
		LabelsDigest:    labelsDigest,
		ChannelID:       0,
		ChannelName:     fmt.Sprintf("（被抑制·源告警=%s）", event.SourceAlertName),
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    fmt.Sprintf("inhibition_suppressed: rule=%s source=%s", event.RuleName, event.SourceFingerprint),
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by source fingerprint: %s", event.SourceFingerprint), s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&e, payload)
	_ = s.db.WithContext(ctx).Create(&e).Error
}

// shouldInhibit 简化的抑制检查入口
func (s *AlertService) shouldInhibit(ctx context.Context, labels map[string]string) (bool, *model.AlertInhibitionEvent) {
	if s.inhibitionSvc == nil {
		return false, nil
	}
	inhibited, event, err := s.inhibitionSvc.CheckInhibition(ctx, labels)
	if err != nil {
		return false, nil
	}
	return inhibited, event
}

// startInhibitionPruner 启动抑制记录清理任务
func (s *AlertService) startInhibitionPruner(ctx context.Context) {
	if s.inhibitionSvc == nil {
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 刷新缓存以保持最新规则
				_ = s.inhibitionSvc.RefreshCache(ctx)
			}
		}
	}()
}
