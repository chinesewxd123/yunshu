package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/promapi"
)

func (s *AlertService) runMonitorRuleEvaluator(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = s.tickMonitorRules(ctx)
		}
	}
}

func (s *AlertService) tickMonitorRules(ctx context.Context) error {
	var rules []model.AlertMonitorRule
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return err
	}
	now := time.Now()
	for i := range rules {
		rule := &rules[i]
		if s.redis != nil {
			if !s.shouldEvalRuleRedis(ctx, rule.ID, rule.EvalIntervalSeconds, now) {
				continue
			}
			lockSec := rule.EvalIntervalSeconds
			if lockSec > 120 {
				lockSec = 120
			}
			if lockSec < 15 {
				lockSec = 15
			}
			if !s.monitorEvalLockAcquire(ctx, rule.ID, lockSec) {
				continue
			}
			func(r *model.AlertMonitorRule) {
				defer s.monitorEvalLockRelease(ctx, r.ID)
				s.evaluateOneMonitorRule(ctx, r)
			}(rule)
			continue
		}
		if !s.shouldEvalRule(rule.ID, rule.EvalIntervalSeconds, now) {
			continue
		}
		s.evaluateOneMonitorRule(ctx, rule)
	}
	return nil
}

func (s *AlertService) shouldEvalRule(ruleID uint, intervalSec int, now time.Time) bool {
	if intervalSec < 5 {
		intervalSec = 5
	}
	iv := time.Duration(intervalSec) * time.Second
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	st, ok := s.monitorEvalState[ruleID]
	if !ok {
		st = &monitorEvalRuleState{}
		s.monitorEvalState[ruleID] = st
	}
	if st.lastEval.IsZero() || now.Sub(st.lastEval) >= iv {
		st.lastEval = now
		return true
	}
	return false
}

func buildMonitorRuleLabels(rule *model.AlertMonitorRule) map[string]string {
	labels := map[string]string{
		"alertname":       rule.Name,
		"severity":        strings.TrimSpace(rule.Severity),
		"monitor_rule_id": fmt.Sprintf("%d", rule.ID),
		"datasource_id":   fmt.Sprintf("%d", rule.DatasourceID),
		"project_id":      fmt.Sprintf("%d", rule.ProjectID),
		"source":          "prometheus_monitor",
	}
	if strings.TrimSpace(rule.Severity) == "" {
		labels["severity"] = "warning"
	}
	raw := strings.TrimSpace(rule.LabelsJSON)
	if raw != "" && raw != "{}" {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for k, v := range obj {
				labels[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return labels
}

func buildMonitorRuleAnnotations(rule *model.AlertMonitorRule) map[string]string {
	ann := map[string]string{
		"summary":     fmt.Sprintf("监控规则 %s 触发", rule.Name),
		"description": fmt.Sprintf("PromQL: %s", rule.Expr),
	}
	raw := strings.TrimSpace(rule.AnnotationsJSON)
	if raw != "" && raw != "{}" {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for k, v := range obj {
				ann[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return ann
}

func (s *AlertService) evaluateOneMonitorRule(ctx context.Context, rule *model.AlertMonitorRule) {
	var ds model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&ds, rule.DatasourceID).Error; err != nil {
		return
	}
	if !ds.Enabled || ds.Type != "prometheus" {
		return
	}
	cli := &promapi.Client{
		BaseURL:       strings.TrimRight(strings.TrimSpace(ds.BaseURL), "/"),
		BearerToken:   ds.BearerToken,
		BasicUser:     ds.BasicUser,
		BasicPassword: ds.BasicPassword,
		SkipTLSVerify: ds.SkipTLSVerify,
	}
	qctx, cancel := context.WithTimeout(ctx, time.Duration(maxInt(3, s.cfg.PromQueryTimeout))*time.Second)
	defer cancel()
	body, _, err := cli.QueryInstant(qctx, strings.TrimSpace(rule.Expr), "")
	if err != nil {
		return
	}
	firing, err := promapi.VectorResultNonEmpty(body)
	if err != nil {
		return
	}
	labels := buildMonitorRuleLabels(rule)
	annotations := buildMonitorRuleAnnotations(rule)
	fp := fmt.Sprintf("monitor_rule_%d", rule.ID)
	now := time.Now()

	if s.redis != nil {
		s.evaluateMonitorRuleWithRedis(ctx, rule, firing, labels, annotations, fp, now)
		return
	}

	s.monitorEvalMu.Lock()
	st, ok := s.monitorEvalState[rule.ID]
	if !ok {
		st = &monitorEvalRuleState{}
		s.monitorEvalState[rule.ID] = st
	}

	if firing {
		if st.activeFiring {
			s.monitorEvalMu.Unlock()
			return
		}
		if st.pendingSince == nil {
			if rule.ForSeconds <= 0 {
				st.activeFiring = true
				s.monitorEvalMu.Unlock()
				_ = s.ReceiveAlertmanager(ctx, AlertManagerPayload{
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
			t := now
			st.pendingSince = &t
			s.monitorEvalMu.Unlock()
			return
		}
		forDur := time.Duration(rule.ForSeconds) * time.Second
		if forDur < 0 {
			forDur = 0
		}
		if now.Sub(*st.pendingSince) >= forDur {
			st.activeFiring = true
			st.pendingSince = nil
			s.monitorEvalMu.Unlock()
			_ = s.ReceiveAlertmanager(ctx, AlertManagerPayload{
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
		s.monitorEvalMu.Unlock()
		return
	}

	if st.activeFiring {
		st.activeFiring = false
		st.pendingSince = nil
		s.monitorEvalMu.Unlock()
		_ = s.ReceiveAlertmanager(ctx, AlertManagerPayload{
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
	st.pendingSince = nil
	s.monitorEvalMu.Unlock()
}
