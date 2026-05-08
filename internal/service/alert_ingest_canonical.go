package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
)

// CanonicalIngressAlert 统一入站模型：由 Alertmanager Webhook 载荷或内置评估路径构造。
type CanonicalIngressAlert struct {
	Source              string
	PayloadReceiver     string
	PayloadStatus       string
	GroupLabels         map[string]string
	CommonLabels        map[string]string
	CommonAnnotations   map[string]string
	Version             string
	ExternalURL         string
	TruncatedAlerts     int
	Alert               AlertManagerAlert
}

// CanonicalAlertsFromAlertmanagerPayload 将 Alertmanager Webhook 形态转为统一入站切片。
func CanonicalAlertsFromAlertmanagerPayload(p AlertManagerPayload) []CanonicalIngressAlert {
	rcv := strings.TrimSpace(p.Receiver)
	src := "alertmanager"
	switch rcv {
	case "platform-monitor":
		src = "platform_monitor"
	case "cloud-expiry":
		src = "cloud_expiry"
	}
	out := make([]CanonicalIngressAlert, 0, len(p.Alerts))
	for i := range p.Alerts {
		out = append(out, CanonicalIngressAlert{
			Source:              src,
			PayloadReceiver:     p.Receiver,
			PayloadStatus:       p.Status,
			GroupLabels:         p.GroupLabels,
			CommonLabels:        p.CommonLabels,
			CommonAnnotations:   p.CommonAnnotations,
			Version:             p.Version,
			ExternalURL:         p.ExternalURL,
			TruncatedAlerts:     p.TruncatedAlerts,
			Alert:               p.Alerts[i],
		})
	}
	return out
}

func (s *AlertService) receiveAlertmanagerPayloadSync(ctx context.Context, payload AlertManagerPayload) error {
	return s.ingestCanonicalAlerts(ctx, CanonicalAlertsFromAlertmanagerPayload(payload))
}

func (s *AlertService) ingestCanonicalAlerts(ctx context.Context, items []CanonicalIngressAlert) error {
	var channels []model.AlertChannel
	if err := s.db.WithContext(ctx).Model(&model.AlertChannel{}).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&channels).Error; err != nil {
		return err
	}
	for _, ca := range items {
		alert := ca.Alert
		labels := mergeStringMap(ca.CommonLabels, alert.Labels)
		dsID, dsName, dsType, pipelineSlug := s.resolveAlertDatasourceMeta(ctx, labels, ca.PayloadReceiver)
		labels["monitor_pipeline"] = pipelineSlug
		if dsID > 0 {
			labels["datasource_id"] = fmt.Sprintf("%d", dsID)
		}
		if strings.TrimSpace(dsName) != "" {
			labels["datasource_name"] = strings.TrimSpace(dsName)
		}
		if strings.TrimSpace(dsType) != "" {
			labels["datasource_type"] = strings.TrimSpace(dsType)
		}
		monitorPipeline := pipelineSlug
		annotations := mergeStringMap(ca.CommonAnnotations, alert.Annotations)
		status := strings.TrimSpace(alert.Status)
		if status == "" {
			status = strings.TrimSpace(ca.PayloadStatus)
		}
		status = strings.ToLower(strings.TrimSpace(status))
		if status == "" {
			status = "firing"
		}
		eventName := strings.TrimSpace(labels["alertname"])
		if eventName == "" {
			eventName = strings.TrimSpace(ca.CommonLabels["alertname"])
		}
		if eventName == "" {
			eventName = "Alertmanager 告警"
		}
		summary := strings.TrimSpace(annotations["summary"])
		if summary == "" {
			summary = strings.TrimSpace(annotations["description"])
		}
		if summary == "" {
			summary = strings.TrimSpace(ca.CommonAnnotations["summary"])
		}
		if summary == "" {
			summary = "Alertmanager webhook message"
		}
		severity := strings.TrimSpace(labels["severity"])
		if severity == "" {
			severity = strings.TrimSpace(ca.CommonLabels["severity"])
		}
		if severity == "" {
			severity = "warning"
		}
		title := eventName

		dims := alertnotify.ExtractDims(labels)
		groupKey := s.computeGroupKey(ca.PayloadReceiver, status, severity, eventName, labels, dims)
		labelsDigest := alertnotify.DigestLabels(labels)
		envLabel := s.resolveAlertEnvironmentLabel(labels, ca.PayloadReceiver, dims, alert.Labels)
		if s.silenceSvc != nil {
			if sid, muted, err := s.silenceSvc.FirstMatchingSilenceID(ctx, labels, time.Now()); err == nil && muted {
				minPayload := map[string]interface{}{
					"labels": labels, "annotations": annotations, "severity": severity, "status": status,
					"receiver": ca.PayloadReceiver, "fingerprint": alert.Fingerprint,
					"groupKey": groupKey, "cluster": envLabel, "labelsDigest": labelsDigest,
					"monitorPipeline": monitorPipeline,
					"datasourceId":    dsID, "datasourceName": dsName, "datasourceType": dsType,
					"source": ca.Source,
				}
				s.logSilenceSuppressed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, sid, minPayload)
				continue
			}
		}

		count, _, _ := s.updateFingerprintState(ctx, alert.Fingerprint, status)

		outgoing := map[string]interface{}{
			"source":          ca.Source,
			"title":           title,
			"summary":         summary,
			"severity":        severity,
			"status":          status,
			"receiver":        ca.PayloadReceiver,
			"fingerprint":     alert.Fingerprint,
			"count":           count,
			"labels":          labels,
			"annotations":     annotations,
			"group_labels":    ca.GroupLabels,
			"am_version":      ca.Version,
			"startsAt":        alert.StartsAt,
			"endsAt":          alert.EndsAt,
			"generatorURL":    alert.GeneratorURL,
			"truncated":       ca.TruncatedAlerts,
			"occurredAt":      time.Now().Format(time.RFC3339),
			"cluster":         envLabel,
			"monitorPipeline": monitorPipeline,
			"datasourceId":    dsID,
			"datasourceName":  dsName,
			"datasourceType":  dsType,
			"groupKey":        groupKey,
			"labelsDigest":    labelsDigest,
		}

		if status == "firing" && s.inhibitionSvc != nil {
			if inhibited, inhEvent := s.CheckInhibition(ctx, labels); inhibited {
				s.logInhibitionEvent(ctx, title, severity, status, envLabel, groupKey, labelsDigest, inhEvent, outgoing)
				_ = s.RecordSourceInhibition(ctx, labels)
				continue
			}
			_ = s.RecordSourceInhibition(ctx, labels)
		}

		if status == "resolved" && s.inhibitionSvc != nil {
			_ = s.ClearSourceInhibition(ctx, labels)
		}

		ctxEnrich, cancelEnrich := context.WithTimeout(ctx, time.Duration(maxInt(1, s.cfg.PromQueryTimeout))*time.Second)
		currentValue := strings.TrimSpace(s.getCachedCurrentValue(ctx, alert.Fingerprint))
		if currentValue == "" && strings.TrimSpace(s.cfg.PrometheusURL) != "" && strings.TrimSpace(alert.GeneratorURL) != "" {
			if v, qerr := s.queryCurrentValueByGeneratorURL(ctxEnrich, alert.GeneratorURL); qerr == nil && strings.TrimSpace(v) != "" {
				currentValue = strings.TrimSpace(v)
				s.setCachedCurrentValue(ctx, alert.Fingerprint, currentValue)
			}
		}
		cancelEnrich()
		if currentValue == "" {
			currentValue = "-"
		}
		outgoing["current"] = currentValue
		if status == "firing" {
			s.enqueuePrometheusEnrich(promEnrichTask{
				Fingerprint:  alert.Fingerprint,
				GeneratorURL: alert.GeneratorURL,
			})
		}
		s.enrichOutgoingProjectName(ctx, outgoing)
		s.enrichAssigneeAndDutyEmails(ctx, outgoing, labels)

		if status == "firing" {
			_ = s.clearResolvedNotificationSent(ctx, alert.Fingerprint)
			shouldSend, reason, aggCount, firstSeen, lastSeen := s.decideFiringGroupTiming(ctx, groupKey, labelsDigest)
			outgoing["agg_count"] = aggCount
			outgoing["agg_first_seen"] = firstSeen
			outgoing["agg_last_seen"] = lastSeen
			if !shouldSend {
				outgoing["suppressed_reason"] = reason
				s.logSuppressedFiringTiming(ctx, title, severity, status, groupKey, labelsDigest, reason, outgoing)
				continue
			}
		}

		subscriptionChannels, matchedPolicyIDs, matchedPolicyNames, subscriptionSilenceSeconds := s.channelIDSetForAlert(ctx, status, labels)
		outgoing["matchedPolicyIds"] = matchedPolicyIDs
		outgoing["matchedPolicyNames"] = matchedPolicyNames
		outgoing["subscription_silence_seconds"] = subscriptionSilenceSeconds
		if len(subscriptionChannels) == 0 {
			s.logNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, "no_policy_matched")
			continue
		}
		if s.shouldSuppressByRouteSilence(ctx, status, groupKey, matchedPolicyIDs, subscriptionSilenceSeconds, labels) {
			s.logSuppressedRouteSilence(ctx, title, severity, status, envLabel, groupKey, labelsDigest, subscriptionSilenceSeconds, outgoing)
			continue
		}
		if status == "resolved" && !s.alertFiringWasDelivered(ctx, alert.Fingerprint) {
			s.logResolvedSuppressedNoPriorFiringDelivery(ctx, title, severity, status, groupKey, labelsDigest, outgoing)
			_ = s.clearFingerprintState(ctx, alert.Fingerprint)
			if s.redis != nil && strings.TrimSpace(alert.Fingerprint) != "" {
				_ = s.redis.Del(ctx, "alert:current:"+strings.TrimSpace(alert.Fingerprint)).Err()
			}
			_ = s.clearGroupAggregateState(ctx, groupKey)
			continue
		}
		if status == "resolved" {
			firstResolved, _ := s.markResolvedNotificationSent(ctx, alert.Fingerprint)
			if !firstResolved {
				outgoing["resolved_sent"] = false
				outgoing["summary"] = "重复恢复事件已抑制（同一告警实例仅发送一次恢复通知）。"
				s.logSuppressedResolvedAggregate(ctx, title, severity, status, groupKey, outgoing)
				continue
			}
			outgoing["resolved_sent"] = true
		}
		sentCount := 0
		okDeliveries := 0
		for i := range channels {
			if _, ok := subscriptionChannels[channels[i].ID]; !ok {
				continue
			}
			settings, _ := parseChannelSettings(channels[i].HeadersJSON)
			if !channelMatchesAlert(settings, labels, dims) {
				continue
			}
			sentCount++
			code, _, err := s.sendToChannel(ctx, &channels[i], alertdispatch.NewEnvelope(ca.Source, title, severity, status, outgoing))
			if err == nil && code >= 200 && code < 300 {
				okDeliveries++
			}
		}
		if status == "firing" && okDeliveries > 0 {
			s.markAlertFiringDelivered(ctx, alert.Fingerprint)
		}
		if status == "resolved" && okDeliveries == 0 {
			_ = s.clearResolvedNotificationSent(ctx, alert.Fingerprint)
		}
		if sentCount == 0 {
			reason := "no_channel_matched"
			if len(channels) == 0 {
				reason = "no_enabled_channels"
			} else if len(subscriptionChannels) > 0 {
				reason = "no_channel_matched_subscription"
			}
			s.logNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, reason)
		}
		if status == "firing" && sentCount > 0 && okDeliveries == 0 {
			s.logAllChannelsDeliveryFailed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing)
		}
		if status == "resolved" {
			_ = s.clearFingerprintState(ctx, alert.Fingerprint)
			if s.redis != nil && strings.TrimSpace(alert.Fingerprint) != "" {
				_ = s.redis.Del(ctx, "alert:current:"+strings.TrimSpace(alert.Fingerprint)).Err()
			}
			_ = s.clearGroupAggregateState(ctx, groupKey)
			s.clearAlertFiringDelivered(ctx, alert.Fingerprint)
		}
	}
	return nil
}
