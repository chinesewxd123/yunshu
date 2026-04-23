package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/promapi"
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
			_ = s.tickCloudExpiryRules(ctx)
		}
	}
}

func (s *AlertService) tickMonitorRules(ctx context.Context) error {
	type evalRule struct {
		model.AlertMonitorRule
		ProjectID uint `gorm:"column:project_id"`
	}
	var rules []evalRule
	if err := s.db.WithContext(ctx).
		Table("alert_monitor_rules amr").
		Select("amr.*, ad.project_id AS project_id").
		Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
		Where("amr.enabled = ?", true).
		Find(&rules).Error; err != nil {
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
			func(r *evalRule) {
				defer s.monitorEvalLockRelease(ctx, r.ID)
				s.evaluateOneMonitorRule(ctx, &r.AlertMonitorRule, r.ProjectID)
			}(rule)
			continue
		}
		if !s.shouldEvalRule(rule.ID, rule.EvalIntervalSeconds, now) {
			continue
		}
		s.evaluateOneMonitorRule(ctx, &rule.AlertMonitorRule, rule.ProjectID)
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

func buildMonitorRuleLabels(rule *model.AlertMonitorRule, projectID uint) map[string]string {
	labels := map[string]string{
		"alertname":       rule.Name,
		"severity":        strings.TrimSpace(rule.Severity),
		"monitor_rule_id": fmt.Sprintf("%d", rule.ID),
		"datasource_id":   fmt.Sprintf("%d", rule.DatasourceID),
		"project_id":      fmt.Sprintf("%d", projectID),
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

func renderRuleAnnotationTemplate(tpl string, labels map[string]string, value string, rule *model.AlertMonitorRule) string {
	out := strings.TrimSpace(tpl)
	if out == "" {
		return ""
	}
	re := regexp.MustCompile(`\{\{\s*\$labels\.([a-zA-Z0-9_]+)\s*\}\}`)
	out = re.ReplaceAllStringFunc(out, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) != 2 {
			return ""
		}
		return strings.TrimSpace(labels[sub[1]])
	})
	out = strings.ReplaceAll(out, "{{$value}}", strings.TrimSpace(value))
	if rule != nil {
		out = strings.ReplaceAll(out, "{{.RuleName}}", strings.TrimSpace(rule.Name))
		out = strings.ReplaceAll(out, "{{.Expr}}", strings.TrimSpace(rule.Expr))
	}
	return out
}

func buildMonitorRuleAnnotations(rule *model.AlertMonitorRule, labels map[string]string, value string) map[string]string {
	defaultSummary := fmt.Sprintf("监控规则 %s 触发", rule.Name)
	defaultDescription := fmt.Sprintf("PromQL: %s", rule.Expr)
	ann := map[string]string{
		"summary":     defaultSummary,
		"description": defaultDescription,
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
	ann["summary"] = renderRuleAnnotationTemplate(ann["summary"], labels, value, rule)
	ann["description"] = renderRuleAnnotationTemplate(ann["description"], labels, value, rule)
	if strings.TrimSpace(ann["summary"]) == "" {
		ann["summary"] = renderRuleAnnotationTemplate(defaultSummary, labels, value, rule)
	}
	if strings.TrimSpace(ann["description"]) == "" {
		ann["description"] = renderRuleAnnotationTemplate(defaultDescription, labels, value, rule)
	}
	return ann
}

func parsePromFirstSample(body []byte) (map[string]string, string) {
	var wrap struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return map[string]string{}, ""
	}
	if wrap.Status != "success" || wrap.Data.ResultType != "vector" || len(wrap.Data.Result) == 0 {
		return map[string]string{}, ""
	}
	first := wrap.Data.Result[0]
	value := ""
	if len(first.Value) >= 2 {
		value = strings.TrimSpace(fmt.Sprintf("%v", first.Value[1]))
	}
	if first.Metric == nil {
		return map[string]string{}, value
	}
	return first.Metric, value
}

func (s *AlertService) evaluateOneMonitorRule(ctx context.Context, rule *model.AlertMonitorRule, projectID uint) {
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
	if projectID == 0 {
		projectID = ds.ProjectID
	}
	labels := buildMonitorRuleLabels(rule, projectID)
	sampleLabels, sampleValue := parsePromFirstSample(body)
	for k, v := range sampleLabels {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		if _, exists := labels[k]; !exists {
			labels[k] = v
		}
	}
	annotations := buildMonitorRuleAnnotations(rule, labels, sampleValue)
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
