package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
)

type groupAggregateSpec struct {
	keyPrefix      string
	titleSuffix    string
	channelName    string
	errorMessage   string
	responsePrefix string
}

func (s *AlertService) updateGroupAggregateStateBySpec(ctx context.Context, spec groupAggregateSpec, groupKey string, interval time.Duration) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
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
	if now.Sub(ls) >= interval {
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
		Source:          "alertmanager",
		Title:           title + spec.titleSuffix,
		Severity:        severity,
		Status:          status,
		Cluster:         strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(groupKey),
		LabelsDigest:    strings.TrimSpace(fmt.Sprintf("%v", payload["labelsDigest"])),
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
		titleSuffix:    " (resolved aggregate suppressed)",
		channelName:    "（未外发·恢复抑制）",
		errorMessage:   "resolved_aggregate_suppressed",
		responsePrefix: "suppressed by resolved once: ",
	}
}

func (s *AlertService) firingGroupAggregateSpec() groupAggregateSpec {
	return groupAggregateSpec{
		keyPrefix: "alert:group:",
		titleSuffix:    " (aggregate suppressed)",
		channelName:    "（未外发·分组节流抑制）",
		errorMessage:   "group_throttled",
		responsePrefix: "suppressed by group timing: ",
	}
}

func (s *AlertService) updateGroupResolvedState(ctx context.Context, groupKey string) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
	// resolved 已改为“同 fingerprint 仅发送一次”，此处仅保留接口形态（不会参与抑制判断）。
	return s.updateGroupAggregateStateBySpec(ctx, s.resolvedGroupAggregateSpec(), groupKey, 0)
}

func (s *AlertService) clearGroupResolvedState(ctx context.Context, groupKey string) error {
	return s.clearGroupAggregateStateBySpec(ctx, s.resolvedGroupAggregateSpec(), groupKey)
}

func (s *AlertService) logSuppressedResolvedAggregate(ctx context.Context, title, severity, status, groupKey string, payload map[string]interface{}) {
	s.logSuppressedGroupAggregateBySpec(ctx, s.resolvedGroupAggregateSpec(), title, severity, status, groupKey, payload)
}

func (s *AlertService) updateGroupAggregateState(ctx context.Context, groupKey string) (shouldSend bool, count int64, firstSeen string, lastSeen string) {
	// 保留旧接口：等价于 repeat_interval_seconds 的简单节流（不含 group_wait/group_interval）。
	return s.updateGroupAggregateStateBySpec(ctx, s.firingGroupAggregateSpec(), groupKey, time.Duration(s.cfg.RepeatIntervalSeconds)*time.Second)
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

// markResolvedNotificationSent 标记该 fingerprint 的恢复通知已发送。
// 返回 true 表示本次首次标记（应发送），false 表示已存在（应抑制重复 resolved 通知）。
func (s *AlertService) markResolvedNotificationSent(ctx context.Context, fingerprint string) (bool, error) {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" {
		return true, nil
	}
	key := "alert:resolved:sent:" + strings.TrimSpace(fingerprint)
	ok, err := s.redis.SetNX(ctx, key, "1", time.Duration(s.cfg.DedupTTLSeconds)*time.Second).Result()
	if err != nil {
		return true, err
	}
	return ok, nil
}

// clearResolvedNotificationSent 在进入新一轮 firing 前清理 resolved 发送标记。
func (s *AlertService) clearResolvedNotificationSent(ctx context.Context, fingerprint string) error {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" {
		return nil
	}
	return s.redis.Del(ctx, "alert:resolved:sent:"+strings.TrimSpace(fingerprint)).Err()
}

func (s *AlertService) logSuppressedDedup(ctx context.Context, title, severity, status, fingerprint string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          "alertmanager",
		Title:           title + " (dedup suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(fmt.Sprintf("%v", payload["groupKey"])),
		LabelsDigest:    strings.TrimSpace(fmt.Sprintf("%v", payload["labelsDigest"])),
		ChannelName:     "（未外发·指纹去重抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    "dedup_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText("suppressed by fingerprint dedup: "+fingerprint, s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}

// decideFiringGroupTiming 对齐 Alertmanager/N9E 的 group_wait/group_interval/repeat_interval 语义（基于 groupKey + labelsDigest）。
//
// - group_wait: 首次见到该 groupKey 后，延迟一段时间再首次发送（收集同组告警）。
// - group_interval: 已发送后，若 labelsDigest 变化（视为“新变化”），至少间隔 group_interval 才可再次发送。
// - repeat_interval: 持续 firing 且无新变化时，按 repeat_interval 重复发送。
//
// 说明：为保持当前架构“单条告警投递”的形态，这里用 labelsDigest 近似表示“组内容变化”。
func (s *AlertService) decideFiringGroupTiming(ctx context.Context, groupKey, labelsDigest string) (shouldSend bool, reason string, count int64, firstSeen string, lastSeen string) {
	if s.redis == nil || strings.TrimSpace(groupKey) == "" {
		return true, "", 1, "", ""
	}
	gk := strings.TrimSpace(groupKey)
	key := "alert:group:timing:" + gk
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, "last_seen", nowStr)
	pipe.HSetNX(ctx, key, "first_seen", nowStr)
	incr := pipe.HIncrBy(ctx, key, "count", 1)
	lastSentCmd := pipe.HGet(ctx, key, "last_sent")
	firstSeenCmd := pipe.HGet(ctx, key, "first_seen")
	lastDigestCmd := pipe.HGet(ctx, key, "last_digest")
	pipe.Expire(ctx, key, time.Duration(s.cfg.AggregateTTLSeconds)*time.Second)
	_, _ = pipe.Exec(ctx)

	c, err := incr.Result()
	if err != nil || c <= 0 {
		c = 1
	}
	fs, _ := firstSeenCmd.Result()
	lsRaw, _ := lastSentCmd.Result()
	lastDigest, _ := lastDigestCmd.Result()

	firstSeen = strings.TrimSpace(fs)
	lastSeen = nowStr

	groupWait := time.Duration(maxInt(0, s.cfg.GroupWaitSeconds)) * time.Second
	groupInterval := time.Duration(maxInt(0, s.cfg.GroupIntervalSeconds)) * time.Second
	repeatInterval := time.Duration(maxInt(1, s.cfg.RepeatIntervalSeconds)) * time.Second

	// 首次发送：受 group_wait 控制
	if strings.TrimSpace(lsRaw) == "" {
		if groupWait > 0 && strings.TrimSpace(firstSeen) != "" {
			if fst, e := time.Parse(time.RFC3339, strings.TrimSpace(firstSeen)); e == nil {
				if now.Sub(fst) < groupWait {
					return false, "group_wait_suppressed", c, firstSeen, lastSeen
				}
			}
		}
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		if strings.TrimSpace(labelsDigest) != "" {
			_ = s.redis.HSet(ctx, key, "last_digest", strings.TrimSpace(labelsDigest)).Err()
		}
		return true, "", c, firstSeen, lastSeen
	}

	lastSent, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(lsRaw))
	if parseErr != nil {
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		if strings.TrimSpace(labelsDigest) != "" {
			_ = s.redis.HSet(ctx, key, "last_digest", strings.TrimSpace(labelsDigest)).Err()
		}
		return true, "", c, firstSeen, lastSeen
	}

	curDigest := strings.TrimSpace(labelsDigest)
	prevDigest := strings.TrimSpace(lastDigest)
	changed := curDigest != "" && prevDigest != "" && curDigest != prevDigest

	// 组有“新变化”：受 group_interval 控制
	if changed {
		if groupInterval > 0 && now.Sub(lastSent) < groupInterval {
			return false, "group_interval_suppressed", c, firstSeen, lastSeen
		}
		_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
		_ = s.redis.HSet(ctx, key, "last_digest", curDigest).Err()
		return true, "", c, firstSeen, lastSeen
	}

	// 无新变化：受 repeat_interval 控制
	if now.Sub(lastSent) < repeatInterval {
		return false, "repeat_suppressed", c, firstSeen, lastSeen
	}
	_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
	if curDigest != "" {
		_ = s.redis.HSet(ctx, key, "last_digest", curDigest).Err()
	}
	return true, "", c, firstSeen, lastSeen
}

func (s *AlertService) logSuppressedFiringTiming(ctx context.Context, title, severity, status, groupKey, labelsDigest, reason string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          "alertmanager",
		Title:           title + " (group timing suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(groupKey),
		LabelsDigest:    strings.TrimSpace(labelsDigest),
		ChannelName:     "（未外发·分组节流抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    strings.TrimSpace(reason),
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText("suppressed by group timing: "+groupKey, s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}
