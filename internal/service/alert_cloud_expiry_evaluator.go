package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
)

// cloudExpiryCronParser 支持五段/六段（可选秒）、以及 @every 等描述符（与 robfig/cron v3 一致）。
var cloudExpiryCronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func parseCloudExpiryCronSchedule(spec string) (cron.Schedule, error) {
	return cloudExpiryCronParser.Parse(strings.TrimSpace(spec))
}

// ValidateCloudExpiryCronSpec 校验云到期规则的 Cron 表达式；空串表示不启用 Cron 调度。
func ValidateCloudExpiryCronSpec(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	if _, err := parseCloudExpiryCronSchedule(spec); err != nil {
		return constants.ErrBadRequestWithMsg("无效的 eval_cron_spec：" + err.Error())
	}
	return nil
}

func shouldEvalCloudExpiryByCron(spec string, last time.Time, hasLast bool, now time.Time) bool {
	sched, err := parseCloudExpiryCronSchedule(spec)
	if err != nil {
		return false
	}
	if !hasLast || last.IsZero() {
		return true
	}
	next := sched.Next(last)
	return !now.Before(next)
}

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
		if !force && !rule.ScheduleEnabled {
			continue
		}
		interval := rule.EvalIntervalSeconds
		if interval <= 0 {
			interval = 3600
		}
		syntheticID := uint(1000000) + rule.ID
		if !force {
			cronSpec := strings.TrimSpace(rule.EvalCronSpec)
			var allow bool
			if cronSpec != "" {
				var last time.Time
				var hasLast bool
				if s.redis != nil {
					last, hasLast = s.redisLastEvalTime(ctx, syntheticID)
				} else {
					last, hasLast = s.cloudExpiryLocalLastEval(syntheticID)
				}
				allow = shouldEvalCloudExpiryByCron(cronSpec, last, hasLast, now)
			} else {
				if s.redis != nil {
					allow = s.shouldEvalRuleRedis(ctx, syntheticID, interval, now)
				} else {
					allow = s.shouldEvalCloudExpiryLocalThrottle(syntheticID, interval, now)
				}
			}
			if !allow {
				continue
			}
		}
		s.evaluateOneCloudExpiryRule(ctx, rule, now, force)
		if !force {
			if s.redis != nil {
				s.redisTouchLastEval(ctx, syntheticID, now)
			} else {
				s.touchCloudExpiryNoRedisLastEval(syntheticID, now)
			}
		}
	}
	return nil
}

// EvaluateCloudExpiryRulesNow 手动触发一次云到期规则评估。
func (s *AlertService) EvaluateCloudExpiryRulesNow(ctx context.Context) error {
	if s.aead == nil {
		return constants.ErrBadRequestWithMsg(
			"未配置 security.encryption_key（或与保存云账号凭据时使用的密钥不一致），无法解密 AK/SK，云到期规则不会拉取云实例。配置密钥后重试「立即评估」。")
	}
	return s.tickCloudExpiryRulesWithMode(ctx, true)
}

func (s *AlertService) evaluateOneCloudExpiryRule(ctx context.Context, rule *model.CloudExpiryRule, now time.Time, manualEval bool) {
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
				if strings.EqualFold(strings.TrimSpace(acc.Provider), "tencent") {
					if !instanceMatchesTencentRegionFilter(region, regionFilter) {
						continue
					}
				} else if _, ok := regionFilter[region]; !ok {
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
			s.emitCloudExpiryAlert(ctx, fp, firing, labels, annotations, now, manualEval)
		}
	}
}

func (s *AlertService) emitCloudExpiryAlert(ctx context.Context, fp string, firing bool, labels, annotations map[string]string, now time.Time, manualEval bool) {
	s.monitorEvalMu.Lock()
	active := s.cloudExpiryState[fp]
	if firing {
		// 定时评估：同一实例已 firing 则不再重复入站。立即评估：始终再入站一次，满足「点击即查即发」。
		if active && !manualEval {
			s.monitorEvalMu.Unlock()
			return
		}
		s.cloudExpiryState[fp] = true
		s.monitorEvalMu.Unlock()
		am := AlertManagerAlert{
			Status:       "firing",
			Labels:       labels,
			Annotations:  annotations,
			StartsAt:     now,
			EndsAt:       now.Add(24 * time.Hour),
			GeneratorURL: "",
			Fingerprint:  fp,
			// 定时/手动云到期均跳过 Redis group_wait，命中阈值后尽快入库与投递。
			SkipGroupTiming: true,
		}
		_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
			Receiver:     "cloud-expiry",
			Status:       "firing",
			GroupLabels:  map[string]string{"alertname": labels["alertname"]},
			CommonLabels: labels,
			Alerts:       []AlertManagerAlert{am},
		})
		return
	}
	if !active {
		s.monitorEvalMu.Unlock()
		return
	}
	delete(s.cloudExpiryState, fp)
	s.monitorEvalMu.Unlock()
	_ = s.receiveAlertmanagerPayloadSync(ctx, AlertManagerPayload{
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

// 无 Redis 时按进程内时间戳节流云到期评估，避免每 5s 全量拉云 API。
func (s *AlertService) cloudExpiryLocalLastEval(syntheticID uint) (time.Time, bool) {
	s.cloudExpiryEvalMu.Lock()
	defer s.cloudExpiryEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		return time.Time{}, false
	}
	last, ok := s.cloudExpiryNoRedisLastEval[syntheticID]
	if !ok || last.IsZero() {
		return time.Time{}, false
	}
	return last, true
}

func (s *AlertService) shouldEvalCloudExpiryLocalThrottle(syntheticID uint, intervalSec int, now time.Time) bool {
	if intervalSec < 60 {
		intervalSec = 60
	}
	s.cloudExpiryEvalMu.Lock()
	defer s.cloudExpiryEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		return true
	}
	last, ok := s.cloudExpiryNoRedisLastEval[syntheticID]
	if !ok || last.IsZero() {
		return true
	}
	return now.Sub(last) >= time.Duration(intervalSec)*time.Second
}

func (s *AlertService) touchCloudExpiryNoRedisLastEval(syntheticID uint, now time.Time) {
	s.cloudExpiryEvalMu.Lock()
	defer s.cloudExpiryEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		s.cloudExpiryNoRedisLastEval = make(map[uint]time.Time)
	}
	s.cloudExpiryNoRedisLastEval[syntheticID] = now
}
