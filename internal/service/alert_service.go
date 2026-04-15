package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-permission-system/internal/config"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/alertnotify"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/mailer"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AlertChannelListQuery struct {
	Keyword string `form:"keyword"`
}

type AlertEventListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Cluster  string `form:"cluster"`
	GroupKey string `form:"group_key"`
}

type AlertChannelUpsertRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Type        string `json:"type"`
	URL         string `json:"url" binding:"omitempty,url,max=1024"`
	Secret      string `json:"secret"`
	HeadersJSON string `json:"headers_json"`
	Enabled     *bool  `json:"enabled"`
	TimeoutMS   int    `json:"timeout_ms"`
}

type AlertTestRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Severity string `json:"severity"`
}

type AlertManagerPayload struct {
	Status            string              `json:"status"`
	Receiver          string              `json:"receiver"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	Alerts            []AlertManagerAlert `json:"alerts"`
}

type AlertManagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

type AlertService struct {
	db          *gorm.DB
	redis       *redis.Client
	mailer      mailer.Sender
	cfg         config.AlertConfig
	enrichQueue chan promEnrichTask
}

type promEnrichTask struct {
	Fingerprint  string
	GeneratorURL string
}

func NewAlertService(db *gorm.DB, redisClient *redis.Client, sender mailer.Sender, cfg config.AlertConfig) *AlertService {
	if cfg.DefaultTimeoutMS <= 0 {
		cfg.DefaultTimeoutMS = 5000
	}
	if cfg.MaxPayloadChars <= 0 {
		cfg.MaxPayloadChars = 8000
	}
	if cfg.DedupTTLSeconds <= 0 {
		cfg.DedupTTLSeconds = 86400
	}
	if cfg.PromQueryTimeout <= 0 {
		cfg.PromQueryTimeout = 5
	}
	if cfg.NotifyIntervalSeconds <= 0 {
		cfg.NotifyIntervalSeconds = 300
	}
	if cfg.ResolvedNotifyIntervalSeconds <= 0 {
		cfg.ResolvedNotifyIntervalSeconds = 30
	}
	if cfg.AggregateTTLSeconds <= 0 {
		cfg.AggregateTTLSeconds = 86400
	}
	if len(cfg.GroupBy) == 0 {
		cfg.GroupBy = []string{"alertname", "cluster", "namespace", "severity", "receiver"}
	}
	if cfg.PlatformLimits.DingdingMaxChars <= 0 {
		cfg.PlatformLimits.DingdingMaxChars = 4500
	}
	if cfg.PlatformLimits.WeComMaxChars <= 0 {
		cfg.PlatformLimits.WeComMaxChars = 3500
	}
	if cfg.PlatformLimits.GenericMaxChars <= 0 {
		cfg.PlatformLimits.GenericMaxChars = 8000
	}
	svc := &AlertService{db: db, redis: redisClient, mailer: sender, cfg: cfg}
	svc.startPrometheusEnrichWorkers()
	return svc
}

func (s *AlertService) startPrometheusEnrichWorkers() {
	if !s.cfg.PrometheusEnrichEnabled {
		return
	}
	if strings.TrimSpace(s.cfg.PrometheusURL) == "" {
		return
	}
	size := s.cfg.PrometheusEnrichQueueSize
	if size <= 0 {
		size = 1024
	}
	workers := s.cfg.PrometheusEnrichWorkers
	if workers <= 0 {
		workers = 4
	}
	s.enrichQueue = make(chan promEnrichTask, size)
	for i := 0; i < workers; i++ {
		go func() {
			for task := range s.enrichQueue {
				if strings.TrimSpace(task.GeneratorURL) == "" || strings.TrimSpace(task.Fingerprint) == "" {
					continue
				}
				val, err := s.queryCurrentValueByGeneratorURL(context.Background(), task.GeneratorURL)
				if err != nil || strings.TrimSpace(val) == "" {
					continue
				}
				s.setCachedCurrentValue(context.Background(), task.Fingerprint, strings.TrimSpace(val))
			}
		}()
	}
}

func (s *AlertService) enqueuePrometheusEnrich(task promEnrichTask) {
	if s.enrichQueue == nil {
		return
	}
	select {
	case s.enrichQueue <- task:
	default:
		// 队列满直接丢弃，避免反压到通知主链路
	}
}

func (s *AlertService) getCachedCurrentValue(ctx context.Context, fingerprint string) string {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" {
		return ""
	}
	v, err := s.redis.Get(ctx, "alert:current:"+strings.TrimSpace(fingerprint)).Result()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func (s *AlertService) setCachedCurrentValue(ctx context.Context, fingerprint, value string) {
	if s.redis == nil || strings.TrimSpace(fingerprint) == "" || strings.TrimSpace(value) == "" {
		return
	}
	ttl := time.Duration(maxInt(s.cfg.DedupTTLSeconds, 3600)) * time.Second
	_ = s.redis.Set(ctx, "alert:current:"+strings.TrimSpace(fingerprint), strings.TrimSpace(value), ttl).Err()
}

func (s *AlertService) ListChannels(ctx context.Context, q AlertChannelListQuery) ([]model.AlertChannel, error) {
	var list []model.AlertChannel
	tx := s.db.WithContext(ctx).Model(&model.AlertChannel{}).Order("id DESC")
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR url LIKE ?", like, like)
	}
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *AlertService) CreateChannel(ctx context.Context, req AlertChannelUpsertRequest) (*model.AlertChannel, error) {
	ch := &model.AlertChannel{
		Name:        strings.TrimSpace(req.Name),
		Type:        strings.TrimSpace(req.Type),
		URL:         strings.TrimSpace(req.URL),
		Secret:      strings.TrimSpace(req.Secret),
		HeadersJSON: strings.TrimSpace(req.HeadersJSON),
		TimeoutMS:   req.TimeoutMS,
	}
	if ch.Type == "" {
		ch.Type = "generic_webhook"
	}
	if requiresWebhookURL(ch.Type, ch.HeadersJSON) && strings.TrimSpace(ch.URL) == "" {
		return nil, apperror.BadRequest("非 email 通道必须填写 webhook URL")
	}
	if ch.TimeoutMS <= 0 {
		ch.TimeoutMS = s.cfg.DefaultTimeoutMS
	}
	if req.Enabled == nil {
		ch.Enabled = true
	} else {
		ch.Enabled = *req.Enabled
	}
	if err := validateHeadersJSON(ch.HeadersJSON); err != nil {
		return nil, err
	}
	if err := validateEmailChannelRecipients(ch.Enabled, ch.Type, ch.HeadersJSON); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(ch).Error; err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *AlertService) UpdateChannel(ctx context.Context, id uint, req AlertChannelUpsertRequest) (*model.AlertChannel, error) {
	var ch model.AlertChannel
	if err := s.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	ch.Name = strings.TrimSpace(req.Name)
	ch.Type = strings.TrimSpace(req.Type)
	ch.URL = strings.TrimSpace(req.URL)
	ch.Secret = strings.TrimSpace(req.Secret)
	ch.HeadersJSON = strings.TrimSpace(req.HeadersJSON)
	if ch.Type == "" {
		ch.Type = "generic_webhook"
	}
	if requiresWebhookURL(ch.Type, ch.HeadersJSON) && strings.TrimSpace(ch.URL) == "" {
		return nil, apperror.BadRequest("非 email 通道必须填写 webhook URL")
	}
	if req.TimeoutMS > 0 {
		ch.TimeoutMS = req.TimeoutMS
	}
	if ch.TimeoutMS <= 0 {
		ch.TimeoutMS = s.cfg.DefaultTimeoutMS
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}
	if err := validateHeadersJSON(ch.HeadersJSON); err != nil {
		return nil, err
	}
	if err := validateEmailChannelRecipients(ch.Enabled, ch.Type, ch.HeadersJSON); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Save(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *AlertService) DeleteChannel(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.AlertChannel{}, id).Error
}

func (s *AlertService) TestChannel(ctx context.Context, id uint, req AlertTestRequest) error {
	var ch model.AlertChannel
	if err := s.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Webhook 测试告警"
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = "这是一条来自 go-permission-system 的测试告警消息。"
	}
	severity := strings.TrimSpace(req.Severity)
	if severity == "" {
		severity = "info"
	}
	payload := map[string]interface{}{
		"source":      "manual-test",
		"title":       title,
		"content":     content,
		"summary":     content,
		"severity":    severity,
		"status":      "firing",
		"occurred_at": time.Now().Format(time.RFC3339),
		"cluster":     "manual-test",
	}
	_, _, err := s.sendToChannel(ctx, &ch, "manual-test", title, severity, "firing", payload)
	return err
}

func (s *AlertService) ListEvents(ctx context.Context, q AlertEventListQuery) (list []model.AlertEvent, total int64, page int, pageSize int, err error) {
	page, pageSize = normalizePage(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertEvent{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("title LIKE ? OR error_message LIKE ? OR channel_name LIKE ?", like, like, like)
	}
	if v := strings.TrimSpace(q.Cluster); v != "" {
		tx = tx.Where("cluster = ?", v)
	}
	if v := strings.TrimSpace(q.GroupKey); v != "" {
		tx = tx.Where("group_key = ?", v)
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	if err = tx.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	return list, total, page, pageSize, nil
}

// 钉钉 markdown：正文里需出现 @手机号 / @userid / @all；企业微信仅用 mentioned_* 即可，勿再拼此段以免双重 @。
func atNotifyPlainMentionsFooter(atMobiles, atUserIds []string, isAtAll bool) string {
	var parts []string
	if isAtAll {
		parts = append(parts, "@all")
	}
	for _, m := range atMobiles {
		m = strings.TrimSpace(m)
		if m != "" {
			parts = append(parts, "@"+m)
		}
	}
	for _, u := range atUserIds {
		u = strings.TrimSpace(u)
		if u != "" {
			parts = append(parts, "@"+u)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(parts, " ")
}

func appendDingTalkMarkdownText(body map[string]interface{}, extra string) {
	if body == nil || strings.TrimSpace(extra) == "" {
		return
	}
	if mm, ok := body["markdown"].(map[string]string); ok {
		nm := map[string]string{}
		for k, v := range mm {
			nm[k] = v
		}
		nm["text"] = nm["text"] + extra
		body["markdown"] = nm
		return
	}
	if mm, ok := body["markdown"].(map[string]interface{}); ok {
		nm := map[string]interface{}{}
		for k, v := range mm {
			nm[k] = v
		}
		prev := strings.TrimSpace(fmt.Sprintf("%v", nm["text"]))
		nm["text"] = prev + extra
		body["markdown"] = nm
	}
}

func (s *AlertService) computeGroupKey(receiver, status, severity, alertname string, labels map[string]string, dims alertnotify.Dims) string {
	fields := s.cfg.GroupBy
	if len(fields) == 0 {
		fields = []string{"alertname", "cluster", "namespace", "severity", "receiver"}
	}
	get := func(k string) string {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "alertname":
			return strings.TrimSpace(alertname)
		case "cluster":
			return strings.TrimSpace(dims.Cluster)
		case "namespace":
			return strings.TrimSpace(dims.Namespace)
		case "severity":
			return strings.TrimSpace(severity)
		case "receiver":
			return strings.TrimSpace(receiver)
		case "status":
			return strings.TrimSpace(status)
		default:
			if labels != nil {
				return strings.TrimSpace(labels[k])
			}
			return ""
		}
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, strings.ToLower(strings.TrimSpace(f))+"="+get(f))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "gk_" + hex.EncodeToString(sum[:8])
}

func channelMatchesAlert(settings map[string]interface{}, labels map[string]string, dims alertnotify.Dims) bool {
	if settings == nil {
		return true
	}
	getLabel := func(k string) string {
		switch k {
		case "cluster":
			return dims.Cluster
		case "namespace":
			return dims.Namespace
		default:
			if labels == nil {
				return ""
			}
			return labels[k]
		}
	}

	// matchLabels: {"cluster":"prod-1","namespace":"kube-system"}
	if raw, ok := settings["matchLabels"]; ok {
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				expected := strings.TrimSpace(fmt.Sprintf("%v", v))
				if expected == "" {
					continue
				}
				actual := strings.TrimSpace(getLabel(k))
				if actual != expected {
					return false
				}
			}
		}
	}
	// matchRegex: {"namespace":"^(kube-system|monitoring)$"}
	if raw, ok := settings["matchRegex"]; ok {
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				pat := strings.TrimSpace(fmt.Sprintf("%v", v))
				if pat == "" {
					continue
				}
				re, err := regexp.Compile(pat)
				if err != nil {
					return false
				}
				if !re.MatchString(strings.TrimSpace(getLabel(k))) {
					return false
				}
			}
		}
	}
	return true
}

func (s *AlertService) ReceiveAlertmanager(ctx context.Context, payload AlertManagerPayload) error {
	var channels []model.AlertChannel
	if err := s.db.WithContext(ctx).Model(&model.AlertChannel{}).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&channels).Error; err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	for _, alert := range payload.Alerts {
		status := strings.TrimSpace(alert.Status)
		if status == "" {
			status = strings.TrimSpace(payload.Status)
		}
		eventName := strings.TrimSpace(alert.Labels["alertname"])
		if eventName == "" {
			eventName = strings.TrimSpace(payload.CommonLabels["alertname"])
		}
		if eventName == "" {
			eventName = "Alertmanager 告警"
		}
		summary := strings.TrimSpace(alert.Annotations["summary"])
		if summary == "" {
			summary = strings.TrimSpace(alert.Annotations["description"])
		}
		if summary == "" {
			summary = strings.TrimSpace(payload.CommonAnnotations["summary"])
		}
		if summary == "" {
			summary = "Alertmanager webhook message"
		}
		severity := strings.TrimSpace(alert.Labels["severity"])
		if severity == "" {
			severity = strings.TrimSpace(payload.CommonLabels["severity"])
		}
		if severity == "" {
			severity = "warning"
		}
		title := eventName

		count, deduped, _ := s.updateFingerprintState(ctx, alert.Fingerprint, status)

		dims := alertnotify.ExtractDims(alert.Labels)
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
		// P3：异步增强，不阻塞通知主链路（缓存 miss 时上面已尽力同步查询）
		if status == "firing" {
			s.enqueuePrometheusEnrich(promEnrichTask{
				Fingerprint:  alert.Fingerprint,
				GeneratorURL: alert.GeneratorURL,
			})
		}
		groupKey := s.computeGroupKey(payload.Receiver, status, severity, eventName, alert.Labels, dims)
		labelsDigest := alertnotify.DigestLabels(alert.Labels)
		envLabel := strings.TrimSpace(dims.Cluster)
		if envLabel == "" {
			envLabel = alertnotify.InferEnvironmentDisplay(alert.Labels, dims)
		}
		outgoing := map[string]interface{}{
			"source":        "alertmanager",
			"title":         title,
			"summary":       summary,
			"severity":      severity,
			"status":        status,
			"receiver":      payload.Receiver,
			"fingerprint":   alert.Fingerprint,
			"count":         count,
			"labels":        alert.Labels,
			"annotations":   alert.Annotations,
			"starts_at":     alert.StartsAt,
			"ends_at":       alert.EndsAt,
			"generator_url": alert.GeneratorURL,
			"current":       currentValue,
			"occurred_at":   time.Now().Format(time.RFC3339),
			"cluster":       envLabel,
			"group_key":     groupKey,
			"labels_digest": labelsDigest,
		}

		// 服务端第二层收敛：同 group_key 在 firing 状态下按 notify_interval 控制发送频率
		if status == "firing" {
			shouldSend, aggCount, firstSeen, lastSeen := s.updateGroupAggregateState(ctx, groupKey)
			outgoing["agg_count"] = aggCount
			outgoing["agg_first_seen"] = firstSeen
			outgoing["agg_last_seen"] = lastSeen
			if !shouldSend {
				s.logSuppressedAggregate(ctx, title, severity, status, groupKey, outgoing)
				continue
			}
		}
		if status == "resolved" {
			// resolved 恢复汇总：短窗口内同 group_key 合并为一条恢复通知
			shouldSend, rCount, firstSeen, lastSeen := s.updateGroupResolvedState(ctx, groupKey)
			outgoing["resolved_count"] = rCount
			outgoing["resolved_first_seen"] = firstSeen
			outgoing["resolved_last_seen"] = lastSeen
			outgoing["summary"] = fmt.Sprintf("恢复汇总：%d 条在 %s ~ %s 已恢复。", rCount, alertnotify.SafeOr(firstSeen, "-"), alertnotify.SafeOr(lastSeen, "-"))
			outgoing["resolved_sent"] = shouldSend
			if !shouldSend {
				s.logSuppressedResolvedAggregate(ctx, title, severity, status, groupKey, outgoing)
				continue
			}
		}

		if status == "firing" && deduped {
			s.logSuppressedDedup(ctx, title, severity, status, alert.Fingerprint, outgoing)
			continue
		}
		for i := range channels {
			settings, _ := parseChannelSettings(channels[i].HeadersJSON)
			if !channelMatchesAlert(settings, alert.Labels, dims) {
				continue
			}
			_, _, _ = s.sendToChannel(ctx, &channels[i], "alertmanager", title, severity, status, outgoing)
		}
		if status == "resolved" {
			_ = s.clearFingerprintState(ctx, alert.Fingerprint)
			if s.redis != nil && strings.TrimSpace(alert.Fingerprint) != "" {
				_ = s.redis.Del(ctx, "alert:current:"+strings.TrimSpace(alert.Fingerprint)).Err()
			}
			// resolved 到来：清理 firing 聚合状态，避免后续误聚合
			_ = s.clearGroupAggregateState(ctx, groupKey)
			// resolved 汇总发送后再清理 resolved 状态；若本次被去抖抑制，则保留用于后续汇总
			if v, ok := outgoing["resolved_sent"].(bool); ok && v {
				_ = s.clearGroupResolvedState(ctx, groupKey)
			}
		}
	}
	return nil
}

func (s *AlertService) ValidateWebhookToken(token string) bool {
	expected := strings.TrimSpace(s.cfg.WebhookToken)
	if expected == "" {
		return true
	}
	return strings.TrimSpace(token) == expected
}
