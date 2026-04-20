package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-permission-system/internal/model"
)

type groupAggregateSpec struct {
	keyPrefix      string
	interval       func() time.Duration
	titleSuffix    string
	channelName    string
	errorMessage   string
	responsePrefix string
}

func (s *AlertService) updateGroupAggregateStateBySpec(ctx context.Context, spec groupAggregateSpec, groupKey string) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
	if s.redis == nil || strings.TrimSpace(groupKey) == "" {
		return true, 1, "", ""
	}
	key := spec.keyPrefix + strings.TrimSpace(groupKey)
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, "last_seen", nowStr)
	pipe.HSetNX(ctx, key, "first_seen", nowStr)
	incr := pipe.HIncrBy(ctx, key, "count", 1)
	lastSentCmd := pipe.HGet(ctx, key, "last_sent")
	pipe.Expire(ctx, key, time.Duration(s.cfg.AggregateTTLSeconds)*time.Second)
	_, _ = pipe.Exec(ctx)
	c, err := incr.Result()
	if err != nil || c <= 0 {
		c = 1
	}
	lastSent, _ := lastSentCmd.Result()
	if c == 1 || strings.TrimSpace(lastSent) == "" {
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		first, _ := s.redis.HGet(ctx, key, "first_seen").Result()
		return true, c, first, nowStr
	}
	ls, err := time.Parse(time.RFC3339, strings.TrimSpace(lastSent))
	if err != nil {
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		first, _ := s.redis.HGet(ctx, key, "first_seen").Result()
		return true, c, first, nowStr
	}
	if now.Sub(ls) >= spec.interval() {
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		first, _ := s.redis.HGet(ctx, key, "first_seen").Result()
		return true, c, first, nowStr
	}
	first, _ := s.redis.HGet(ctx, key, "first_seen").Result()
	return false, c, first, nowStr
}

func (s *AlertService) clearGroupAggregateStateBySpec(ctx context.Context, spec groupAggregateSpec, groupKey string) error {
	if s.redis == nil || strings.TrimSpace(groupKey) == "" {
		return nil
	}
	return s.redis.Del(ctx, spec.keyPrefix+strings.TrimSpace(groupKey)).Err()
}

func (s *AlertService) logSuppressedGroupAggregateBySpec(ctx context.Context, spec groupAggregateSpec, title, severity, status, groupKey string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             "alertmanager",
		Title:              title + spec.titleSuffix,
		Severity:           severity,
		Status:             status,
		Cluster:            strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline:    strings.TrimSpace(fmt.Sprintf("%v", payload["monitor_pipeline"])),
		GroupKey:           strings.TrimSpace(groupKey),
		LabelsDigest:       strings.TrimSpace(fmt.Sprintf("%v", payload["labels_digest"])),
		ChannelName:     spec.channelName,
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    spec.errorMessage,
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(spec.responsePrefix+groupKey, s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) resolvedGroupAggregateSpec() groupAggregateSpec {
	return groupAggregateSpec{
		keyPrefix: "alert:group:resolved:",
		interval: func() time.Duration {
			return time.Duration(s.cfg.ResolvedNotifyIntervalSeconds) * time.Second
		},
		titleSuffix:    " (resolved aggregate suppressed)",
		channelName:    "（未外发·恢复聚合抑制）",
		errorMessage:   "resolved_aggregate_suppressed",
		responsePrefix: "suppressed by resolved groupKey aggregate: ",
	}
}

func (s *AlertService) firingGroupAggregateSpec() groupAggregateSpec {
	return groupAggregateSpec{
		keyPrefix: "alert:group:",
		interval: func() time.Duration {
			return time.Duration(s.cfg.NotifyIntervalSeconds) * time.Second
		},
		titleSuffix:    " (aggregate suppressed)",
		channelName:    "（未外发·group_key 聚合抑制）",
		errorMessage:   "aggregate_suppressed",
		responsePrefix: "suppressed by groupKey aggregate: ",
	}
}

func (s *AlertService) updateGroupResolvedState(ctx context.Context, groupKey string) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
	return s.updateGroupAggregateStateBySpec(ctx, s.resolvedGroupAggregateSpec(), groupKey)
}

func (s *AlertService) clearGroupResolvedState(ctx context.Context, groupKey string) error {
	return s.clearGroupAggregateStateBySpec(ctx, s.resolvedGroupAggregateSpec(), groupKey)
}

func (s *AlertService) logSuppressedResolvedAggregate(ctx context.Context, title, severity, status, groupKey string, payload map[string]interface{}) {
	s.logSuppressedGroupAggregateBySpec(ctx, s.resolvedGroupAggregateSpec(), title, severity, status, groupKey, payload)
}

func (s *AlertService) updateGroupAggregateState(ctx context.Context, groupKey string) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
	return s.updateGroupAggregateStateBySpec(ctx, s.firingGroupAggregateSpec(), groupKey)
}

func (s *AlertService) clearGroupAggregateState(ctx context.Context, groupKey string) error {
	return s.clearGroupAggregateStateBySpec(ctx, s.firingGroupAggregateSpec(), groupKey)
}

func (s *AlertService) logSuppressedAggregate(ctx context.Context, title, severity, status, groupKey string, payload map[string]interface{}) {
	s.logSuppressedGroupAggregateBySpec(ctx, s.firingGroupAggregateSpec(), title, severity, status, groupKey, payload)
}

func (s *AlertService) updateFingerprintState(ctx context.Context, fingerprint, status string) (count int64, deduped bool, err error) {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" {
		return 1, false, nil
	}
	key := "alert:fingerprint:" + strings.TrimSpace(fingerprint)
	if strings.EqualFold(status, "firing") {
		count, err = s.redis.HIncrBy(ctx, key, "count", 1).Result()
		if err != nil {
			return 1, false, err
		}
		_, _ = s.redis.HSet(ctx, key, "last_status", "firing").Result()
		_, _ = s.redis.Expire(ctx, key, time.Duration(s.cfg.DedupTTLSeconds)*time.Second).Result()
		return count, count > 1, nil
	}
	v, e := s.redis.HGet(ctx, key, "count").Result()
	if e != nil {
		return 1, false, nil
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	if n <= 0 {
		n = 1
	}
	return n, false, nil
}

func (s *AlertService) clearFingerprintState(ctx context.Context, fingerprint string) error {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" {
		return nil
	}
	return s.redis.Del(ctx, "alert:fingerprint:"+strings.TrimSpace(fingerprint)).Err()
}

func (s *AlertService) logSuppressedDedup(ctx context.Context, title, severity, status, fingerprint string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             "alertmanager",
		Title:              title + " (dedup suppressed)",
		Severity:           severity,
		Status:             status,
		Cluster:            strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline:    strings.TrimSpace(fmt.Sprintf("%v", payload["monitor_pipeline"])),
		GroupKey:           strings.TrimSpace(fmt.Sprintf("%v", payload["group_key"])),
		LabelsDigest:       strings.TrimSpace(fmt.Sprintf("%v", payload["labels_digest"])),
		ChannelName:        "（未外发·指纹去重抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    "dedup_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText("suppressed by fingerprint dedup: "+fingerprint, s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}
