package service

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// evaluateMonitorRuleNoRedis 在无 Redis 时做简化状态机（进程内），避免监控规则完全不评估。
func (s *AlertService) evaluateMonitorRuleNoRedis(ctx context.Context, rule *model.AlertMonitorRule, firing bool, labels, annotations map[string]string, fp string, now time.Time) {
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	if s.monitorNoRedisActive == nil {
		s.monitorNoRedisActive = make(map[string]bool)
	}
	wasActive := s.monitorNoRedisActive[fp]

	if firing {
		if !wasActive {
			s.monitorNoRedisActive[fp] = true
			s.emitMonitorPlatformFiring(ctx, rule, labels, annotations, fp, now)
		}
		return
	}
	if wasActive {
		delete(s.monitorNoRedisActive, fp)
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
				Fingerprint:  fp,
			}},
		})
	}
}

func (s *AlertService) shouldEvalRuleNoRedis(ruleID uint, intervalSec int, now time.Time) bool {
	if intervalSec < 5 {
		intervalSec = 5
	}
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		s.cloudExpiryNoRedisLastEval = make(map[uint]time.Time)
	}
	last, ok := s.cloudExpiryNoRedisLastEval[ruleID]
	if !ok || now.Sub(last) >= time.Duration(intervalSec)*time.Second {
		s.cloudExpiryNoRedisLastEval[ruleID] = now
		return true
	}
	return false
}
