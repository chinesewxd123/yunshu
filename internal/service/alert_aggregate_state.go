package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"gorm.io/gorm/clause"
)

type groupAggregateSpec struct {
	keyPrefix      string
	titleSuffix    string
	channelName    string
	errorMessage   string
	responsePrefix string
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
		Source:          alertEventSourceFromPayload(payload),
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
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) resolvedGroupAggregateSpec() groupAggregateSpec {
	return groupAggregateSpec{
		keyPrefix:      "alert:group:resolved:",
		titleSuffix:    "（恢复通知：已合并）",
		channelName:    "（未推送·恢复仅通知一次）",
		errorMessage:   "resolved_aggregate_suppressed",
		responsePrefix: "同一告警实例的恢复通知已合并，本轮未重复推送：",
	}
}

func (s *AlertService) firingGroupAggregateSpec() groupAggregateSpec {
	return groupAggregateSpec{
		keyPrefix:      "alert:group:",
		titleSuffix:    "（通知聚合：本轮未推送）",
		channelName:    "（未推送·聚合限流）",
		errorMessage:   "group_throttled",
		responsePrefix: "按聚合间隔控制本轮未推送：",
	}
}

func (s *AlertService) clearGroupResolvedState(ctx context.Context, groupKey string) error {
	return s.clearGroupAggregateStateBySpec(ctx, s.resolvedGroupAggregateSpec(), groupKey)
}

func (s *AlertService) logSuppressedResolvedAggregate(ctx context.Context, title, severity, status, groupKey string, payload map[string]interface{}) {
	s.logSuppressedGroupAggregateBySpec(ctx, s.resolvedGroupAggregateSpec(), title, severity, status, groupKey, payload)
}

func (s *AlertService) clearGroupAggregateState(ctx context.Context, groupKey string) error {
	return s.clearGroupAggregateStateBySpec(ctx, s.firingGroupAggregateSpec(), groupKey)
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

func firingGroupTimingRedisKey(groupKey string) string {
	return "alert:group:timing:" + strings.TrimSpace(groupKey)
}

// evaluateFiringGroupTiming 纯判定：不写入 last_sent / last_digest（须在通道发送成功后 commit）。
func evaluateFiringGroupTiming(cfg config.AlertConfig, now time.Time, firstSeen, lastSentRaw, lastDigest, labelsDigest string) (shouldSend bool, reason string) {
	groupWait := time.Duration(maxInt(0, cfg.GroupWaitSeconds)) * time.Second
	groupInterval := time.Duration(maxInt(0, cfg.GroupIntervalSeconds)) * time.Second
	repeatInterval := time.Duration(maxInt(1, cfg.RepeatIntervalSeconds)) * time.Second

	if strings.TrimSpace(lastSentRaw) == "" {
		if groupWait > 0 && strings.TrimSpace(firstSeen) != "" {
			if fst, e := time.Parse(time.RFC3339, strings.TrimSpace(firstSeen)); e == nil {
				if now.Sub(fst) < groupWait {
					return false, "group_wait_suppressed"
				}
			}
		}
		return true, ""
	}

	lastSent, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(lastSentRaw))
	if parseErr != nil {
		return true, ""
	}

	curDigest := strings.TrimSpace(labelsDigest)
	prevDigest := strings.TrimSpace(lastDigest)
	changed := curDigest != "" && prevDigest != "" && curDigest != prevDigest

	if changed {
		if groupInterval > 0 && now.Sub(lastSent) < groupInterval {
			return false, "group_interval_suppressed"
		}
		return true, ""
	}

	if now.Sub(lastSent) < repeatInterval {
		return false, "repeat_suppressed"
	}
	return true, ""
}

// peekFiringGroupTiming 只读判定分组节流（更新观测字段 count/first_seen/last_seen，不写 last_sent）。
func (s *AlertService) peekFiringGroupTiming(ctx context.Context, groupKey, labelsDigest string) (shouldSend bool, reason string, count int64, firstSeen string, lastSeen string) {
	if s.redis == nil || strings.TrimSpace(groupKey) == "" {
		return true, "", 1, "", ""
	}
	key := firingGroupTimingRedisKey(groupKey)
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
	shouldSend, reason = evaluateFiringGroupTiming(s.cfg, now, firstSeen, lsRaw, lastDigest, labelsDigest)
	return shouldSend, reason, c, firstSeen, lastSeen
}

// commitFiringGroupTimingSend 在至少一个通道 HTTP 发送成功后调用，记录「上次成功通知」时间。
func (s *AlertService) commitFiringGroupTimingSend(ctx context.Context, groupKey, labelsDigest string) {
	if s.redis == nil || strings.TrimSpace(groupKey) == "" {
		return
	}
	key := firingGroupTimingRedisKey(groupKey)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	_ = s.redis.HSet(ctx, key, "last_sent", nowStr).Err()
	if d := strings.TrimSpace(labelsDigest); d != "" {
		_ = s.redis.HSet(ctx, key, "last_digest", d).Err()
	}
	_ = s.redis.Expire(ctx, key, time.Duration(s.cfg.AggregateTTLSeconds)*time.Second).Err()
}

// decideFiringGroupTiming 兼容旧调用：等价于 peek（不再预写 last_sent）。
func (s *AlertService) decideFiringGroupTiming(ctx context.Context, groupKey, labelsDigest string) (shouldSend bool, reason string, count int64, firstSeen string, lastSeen string) {
	return s.peekFiringGroupTiming(ctx, groupKey, labelsDigest)
}

func firingDeliveredRedisKey(fingerprint string) string {
	return "alert:firing_delivered:" + strings.TrimSpace(fingerprint)
}

// markAlertFiringDelivered 记录「该指纹已成功外发过至少一次 firing 通知」，
// 避免 firing 被分组节流等抑制后，resolved 仍外发造成「只有恢复没有触发」。
func (s *AlertService) markAlertFiringDelivered(ctx context.Context, fingerprint string) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return
	}
	now := time.Now().UTC()
	row := model.AlertFiringDelivery{Fingerprint: fp, UpdatedAt: now}
	_ = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fingerprint"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&row).Error
	if s.redis != nil {
		ttlSec := maxInt(s.cfg.AggregateTTLSeconds, 7*24*3600)
		_ = s.redis.Set(ctx, firingDeliveredRedisKey(fp), "1", time.Duration(ttlSec)*time.Second).Err()
	}
}

func (s *AlertService) alertFiringWasDelivered(ctx context.Context, fingerprint string) bool {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return false
	}
	if s.redis != nil {
		v, err := s.redis.Get(ctx, firingDeliveredRedisKey(fp)).Result()
		if err == nil && strings.TrimSpace(v) == "1" {
			return true
		}
	}
	var n int64
	_ = s.db.WithContext(ctx).Model(&model.AlertFiringDelivery{}).Where("fingerprint = ?", fp).Count(&n).Error
	return n > 0
}

func (s *AlertService) clearAlertFiringDelivered(ctx context.Context, fingerprint string) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return
	}
	if s.redis != nil {
		_ = s.redis.Del(ctx, firingDeliveredRedisKey(fp)).Err()
	}
	_ = s.db.WithContext(ctx).Where("fingerprint = ?", fp).Delete(&model.AlertFiringDelivery{}).Error
}

func (s *AlertService) logResolvedSuppressedNoPriorFiringDelivery(ctx context.Context, title, severity, status, groupKey, labelsDigest string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             alertEventSourceFromPayload(payload),
		Title:              title + " (resolved suppressed: no prior firing delivery)",
		Severity:           severity,
		Status:             status,
		Cluster:            strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline:    strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:           strings.TrimSpace(groupKey),
		LabelsDigest:       strings.TrimSpace(labelsDigest),
		MatchedPolicyIDs:   strings.TrimSpace(fmt.Sprintf("%v", payload["matchedPolicyIds"])),
		MatchedPolicyNames: strings.TrimSpace(fmt.Sprintf("%v", payload["matchedPolicyNames"])),
		ChannelID:          0,
		ChannelName:        "（未外发·无成功触发投递）",
		Success:            true,
		HTTPStatusCode:     200,
		ErrorMessage:       "resolved_no_prior_firing_delivery",
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload:    "firing 未成功投递到任何通道（可能为分组节流抑制或通道失败），已抑制恢复外发",
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func humanReadableGroupTimingSuppression(reason string, cfg config.AlertConfig) string {
	gw := cfg.GroupWaitSeconds
	if gw < 0 {
		gw = 0
	}
	gi := cfg.GroupIntervalSeconds
	if gi < 0 {
		gi = 0
	}
	ri := cfg.RepeatIntervalSeconds
	if ri <= 0 {
		ri = 300
	}
	switch strings.TrimSpace(reason) {
	case "repeat_suppressed":
		return fmt.Sprintf("告警仍处于触发状态。为避免短时间内重复打扰，平台按「重复提醒间隔」（当前 %d 秒）控制：距上次成功通知未满该间隔时，本轮不向各渠道再次推送；本条为留痕记录。", ri)
	case "group_interval_suppressed":
		return fmt.Sprintf("本次告警的标签摘要与上次已发送的不同（例如实例或采样标签变化）。平台按「同组变化间隔」（当前 %d 秒）控制：未满间隔时本轮不推送；达到间隔后会再评估是否发送。", gi)
	case "group_wait_suppressed":
		return fmt.Sprintf("平台按「首次同组等待」（当前 %d 秒）聚合同组告警，等待窗口结束前本轮不推送。", gw)
	default:
		return "本轮根据通知合并策略未向渠道推送，可能与「首次同组等待」「同组变化间隔」或「重复提醒间隔」有关，具体以平台告警配置为准。"
	}
}

func (s *AlertService) logSuppressedFiringTiming(ctx context.Context, title, severity, status, groupKey, labelsDigest, reason string, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          alertEventSourceFromPayload(payload),
		Title:           title + "（通知合并：本轮未推送）",
		Severity:        severity,
		Status:          status,
		Cluster:         strings.TrimSpace(fmt.Sprintf("%v", payload["cluster"])),
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(groupKey),
		LabelsDigest:    strings.TrimSpace(labelsDigest),
		ChannelName:     "（未推送·合并降噪）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    strings.TrimSpace(reason),
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(humanReadableGroupTimingSuppression(reason, s.cfg), s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}
