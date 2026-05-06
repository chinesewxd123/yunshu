package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"yunshu/internal/model"
	cryptox "yunshu/internal/pkg/crypto"
)

func (s *AlertService) tickCloudExpiryRules(ctx context.Context) error {
	return s.tickCloudExpiryRulesWithMode(ctx, false)
}

func (s *AlertService) tickCloudExpiryRulesWithMode(ctx context.Context, force bool) error {
	var rules []model.CloudExpiryRule
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return err
	}
	now := time.Now()
	for i := range rules {
		rule := &rules[i]
		interval := rule.EvalIntervalSeconds
		if interval <= 0 {
			interval = 3600
		}
		syntheticID := uint(1000000) + rule.ID
		if !force && !s.shouldEvalRuleRedis(ctx, syntheticID, interval, now) {
			continue
		}
		s.evaluateOneCloudExpiryRule(ctx, rule, now)
	}
	return nil
}

// EvaluateCloudExpiryRulesNow 手动触发一次云到期规则评估。
func (s *AlertService) EvaluateCloudExpiryRulesNow(ctx context.Context) error {
	return s.tickCloudExpiryRulesWithMode(ctx, true)
}

func (s *AlertService) evaluateOneCloudExpiryRule(ctx context.Context, rule *model.CloudExpiryRule, now time.Time) {
	if s.aead == nil {
		return
	}
	providerFilter := strings.TrimSpace(rule.Provider)
	regionFilter := parseRegionSet(rule.RegionScope)
	var accounts []model.CloudAccount
	tx := s.db.WithContext(ctx).Where("project_id = ? AND status = ?", rule.ProjectID, model.StatusEnabled)
	if providerFilter != "" {
		tx = tx.Where("provider = ?", providerFilter)
	}
	if err := tx.Find(&accounts).Error; err != nil {
		return
	}
	for i := range accounts {
		acc := &accounts[i]
		if acc.EncAK == nil || acc.EncSK == nil {
			continue
		}
		ak, err := cryptox.DecryptString(s.aead, *acc.EncAK)
		if err != nil {
			continue
		}
		sk, err := cryptox.DecryptString(s.aead, *acc.EncSK)
		if err != nil {
			continue
		}
		provider, err := cloudProviderByName(strings.TrimSpace(acc.Provider))
		if err != nil {
			continue
		}
		scope := strings.TrimSpace(acc.RegionScope)
		if ruleScope := strings.TrimSpace(rule.RegionScope); ruleScope != "" {
			scope = ruleScope
		}
		instances, err := provider.ListInstances(ctx, ak, sk, scope)
		if err != nil {
			continue
		}
		for _, ins := range instances {
			instanceID := strings.TrimSpace(ins.InstanceID)
			if instanceID == "" {
				continue
			}
			region := strings.TrimSpace(ins.Region)
			if len(regionFilter) > 0 {
				if _, ok := regionFilter[region]; !ok {
					continue
				}
			}
			expireAt, err := provider.QueryInstanceExpireAt(ctx, ak, sk, region, instanceID)
			if err != nil || expireAt == nil {
				continue
			}
			daysLeft := int(math.Ceil(expireAt.Sub(now).Hours() / 24))
			firing := daysLeft <= maxInt(1, rule.AdvanceDays)
			fp := fmt.Sprintf("cloud_expiry_rule_%d_%s", rule.ID, instanceID)
			labels := map[string]string{
				"alertname":        strings.TrimSpace(rule.Name),
				"severity":         strings.TrimSpace(rule.Severity),
				"source":           "cloud_expiry",
				"project_id":       fmt.Sprintf("%d", rule.ProjectID),
				"provider":         strings.TrimSpace(acc.Provider),
				"cloud_account_id": fmt.Sprintf("%d", acc.ID),
				"instance_id":      instanceID,
				"instance_name":    strings.TrimSpace(ins.Name),
				"region":           region,
			}
			if labels["severity"] == "" {
				labels["severity"] = "warning"
			}
			if raw := strings.TrimSpace(rule.LabelsJSON); raw != "" && raw != "{}" {
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					for k, v := range obj {
						labels[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
					}
				}
			}
			annotations := map[string]string{
				"summary":     fmt.Sprintf("云服务器到期提醒：%s/%s 剩余 %d 天", strings.TrimSpace(acc.Provider), instanceID, daysLeft),
				"description": fmt.Sprintf("实例=%s(%s)，区域=%s，到期时间=%s，剩余天数=%d", strings.TrimSpace(ins.Name), instanceID, region, expireAt.Format(time.RFC3339), daysLeft),
			}
			s.emitCloudExpiryAlert(ctx, fp, firing, labels, annotations, now)
		}
	}
}

func (s *AlertService) emitCloudExpiryAlert(ctx context.Context, fp string, firing bool, labels, annotations map[string]string, now time.Time) {
	s.monitorEvalMu.Lock()
	active := s.cloudExpiryState[fp]
	if firing {
		if active {
			s.monitorEvalMu.Unlock()
			return
		}
		s.cloudExpiryState[fp] = true
		s.monitorEvalMu.Unlock()
		_ = s.ReceiveAlertmanager(ctx, AlertManagerPayload{
			Receiver:     "cloud-expiry",
			Status:       "firing",
			GroupLabels:  map[string]string{"alertname": labels["alertname"]},
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
	if !active {
		s.monitorEvalMu.Unlock()
		return
	}
	delete(s.cloudExpiryState, fp)
	s.monitorEvalMu.Unlock()
	_ = s.ReceiveAlertmanager(ctx, AlertManagerPayload{
		Receiver:     "cloud-expiry",
		Status:       "resolved",
		GroupLabels:  map[string]string{"alertname": labels["alertname"]},
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
}

func cloudProviderByName(name string) (CloudProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alibaba":
		return &AlibabaCloudProvider{}, nil
	case "tencent":
		return &TencentCloudProvider{}, nil
	case "jd":
		return &JdCloudProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider")
	}
}

func parseRegionSet(scope string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, it := range strings.Split(scope, ",") {
		v := strings.TrimSpace(it)
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}
