package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"

	"github.com/redis/go-redis/v9"
)

func monitorEvalStateKey(ruleID uint) string {
	return fmt.Sprintf("alert:mon:state:%d", ruleID)
}

func monitorEvalLockKey(ruleID uint) string {
	return fmt.Sprintf("alert:mon:lock:%d", ruleID)
}

func (s *AlertService) monitorEvalLockAcquire(ctx context.Context, ruleID uint, ttlSec int) bool {
	if s.redis == nil {
		return true
	}
	if ttlSec < 10 {
		ttlSec = 10
	}
	ok, err := s.redis.SetNX(ctx, monitorEvalLockKey(ruleID), "1", time.Duration(ttlSec)*time.Second).Result()
	return err == nil && ok
}

func (s *AlertService) monitorEvalLockRelease(ctx context.Context, ruleID uint) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, monitorEvalLockKey(ruleID)).Err()
}

// redisLastEvalTime 读取上次评估时间（RFC3339Nano），无记录或解析失败时 has=false。
func (s *AlertService) redisLastEvalTime(ctx context.Context, ruleID uint) (t time.Time, has bool) {
	if s.redis == nil {
		return time.Time{}, false
	}
	last, err := s.redis.HGet(ctx, monitorEvalStateKey(ruleID), "last_eval").Result()
	if err != nil && err != redis.Nil {
		return time.Time{}, false
	}
	if strings.TrimSpace(last) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, last)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, last)
		if err != nil {
			return time.Time{}, false
		}
	}
	return parsed, true
}

func (s *AlertService) shouldEvalRuleRedis(ctx context.Context, ruleID uint, intervalSec int, now time.Time) bool {
	if s.redis == nil {
		return true
	}
	if intervalSec < 5 {
		intervalSec = 5
	}
	last, err := s.redis.HGet(ctx, monitorEvalStateKey(ruleID), "last_eval").Result()
	if err != nil && err != redis.Nil {
		return true
	}
	if strings.TrimSpace(last) == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, last)
	if err != nil {
		t, err = time.Parse(time.RFC3339, last)
		if err != nil {
			return true
		}
	}
	return now.Sub(t) >= time.Duration(intervalSec)*time.Second
}

func (s *AlertService) redisTouchLastEval(ctx context.Context, ruleID uint, now time.Time) {
	if s.redis == nil {
		return
	}
	_ = s.redis.HSet(ctx, monitorEvalStateKey(ruleID), "last_eval", now.UTC().Format(time.RFC3339Nano)).Err()
	_ = s.redis.Expire(ctx, monitorEvalStateKey(ruleID), 7*24*time.Hour).Err()
}

func parseRFC3339Ptr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *AlertService) evaluateMonitorRuleWithRedis(ctx context.Context, rule *model.AlertMonitorRule, firing bool, labels map[string]string, annotations map[string]string, fp string, now time.Time) {
	defer s.redisTouchLastEval(ctx, rule.ID, now)

	key := monitorEvalStateKey(rule.ID)
	h, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return
	}
	active := strings.TrimSpace(h["active_firing"]) == "1"
	pendingStr := strings.TrimSpace(h["pending_since"])
	pendingSince, _ := parseRFC3339Ptr(pendingStr)

	if firing {
		if active {
			// 持续 firing：仍需按评估周期入站；否则首轮已将 active_firing=1，
			// 若那次 ingest 未匹配订阅或通道发送失败，将永远不再 retry，直至 PromQL 变非 firing。
			// 重复通知由 ingest 的 group_wait / group_interval / repeat_interval 控制。
			_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
				Receiver:     "platform-monitor",
				Status:       "firing",
				GroupLabels:  map[string]string{"alertname": rule.Name},
				CommonLabels: labels,
				Alerts: []AlertManagerAlert{{
					Status:       "firing",
					Labels:       labels,
					Annotations:  annotations,
					StartsAt:     now,
					EndsAt:       now.Add(24 * time.Hour),
					GeneratorURL: "",
					Fingerprint:  fp,
				}},
			})
			return
		}
		if pendingSince == nil {
			if rule.ForSeconds <= 0 {
				_ = s.redis.HSet(ctx, key, map[string]interface{}{
					"active_firing": "1",
					"pending_since": "",
				}).Err()
				_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
				_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
					Receiver:     "platform-monitor",
					Status:       "firing",
					GroupLabels:  map[string]string{"alertname": rule.Name},
					CommonLabels: labels,
					Alerts: []AlertManagerAlert{{
						Status:       "firing",
						Labels:       labels,
						Annotations:  annotations,
						StartsAt:     now,
						EndsAt:       now.Add(24 * time.Hour),
						GeneratorURL: "",
						Fingerprint:  fp,
					}},
				})
				return
			}
			t := now.UTC()
			_ = s.redis.HSet(ctx, key, "pending_since", t.Format(time.RFC3339Nano)).Err()
			_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
			return
		}
		forDur := time.Duration(rule.ForSeconds) * time.Second
		if forDur < 0 {
			forDur = 0
		}
		if now.Sub(*pendingSince) >= forDur {
			_ = s.redis.HSet(ctx, key, map[string]interface{}{
				"active_firing": "1",
				"pending_since": "",
			}).Err()
			_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
				Receiver:     "platform-monitor",
				Status:       "firing",
				GroupLabels:  map[string]string{"alertname": rule.Name},
				CommonLabels: labels,
				Alerts: []AlertManagerAlert{{
					Status:       "firing",
					Labels:       labels,
					Annotations:  annotations,
					StartsAt:     now,
					EndsAt:       now.Add(24 * time.Hour),
					GeneratorURL: "",
					Fingerprint:  fp,
				}},
			})
			return
		}
		return
	}

	if active {
		_ = s.redis.HSet(ctx, key, map[string]interface{}{
			"active_firing": "0",
			"pending_since": "",
		}).Err()
		_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
			Receiver:     "platform-monitor",
			Status:       "resolved",
			GroupLabels:  map[string]string{"alertname": rule.Name},
			CommonLabels: labels,
			Alerts: []AlertManagerAlert{{
				Status:       "resolved",
				Labels:       labels,
				Annotations:  annotations,
				StartsAt:     now.Add(-time.Minute),
				EndsAt:       now,
				GeneratorURL: "",
				Fingerprint:  fp,
			}},
		})
		return
	}
	_ = s.redis.HSet(ctx, key, "pending_since", "").Err()
}
