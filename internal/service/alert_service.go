package service

import (
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/pagination"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 监控链路：与 Prometheus cluster 标签解耦，便于筛选与策略匹配（避免「同一规则双路告警」难以区分）。
const (
	monitorPipelinePrometheus = "prometheus" // Prometheus.yml + rules + Alertmanager -> 平台 Webhook
	monitorPipelinePlatform   = "platform"   // 平台内 PromQL 规则评估（receiver=platform-monitor）
)

type AlertChannelListQuery struct {
	Keyword string `form:"keyword"`
}

type AlertEventListQuery struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	Keyword         string `form:"keyword"`
	Cluster         string `form:"cluster"`
	MonitorPipeline string `form:"monitor_pipeline"`
	GroupKey        string `form:"group_key"`
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
	Version           string              `json:"version"`
	Receiver          string              `json:"receiver"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	TruncatedAlerts   int                 `json:"truncatedAlerts"`
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
	policySvc   *AlertPolicyService

	silenceSvc  *AlertSilenceService
	assigneeSvc *AlertRuleAssigneeService
	dutySvc     *AlertDutyService

	monitorEvalCancel context.CancelFunc
	monitorEvalMu     sync.Mutex
	monitorEvalState  map[uint]*monitorEvalRuleState
	aead              cipher.AEAD
	cloudExpiryState  map[string]bool
}

type monitorEvalRuleState struct {
	activeFiring bool
	pendingSince *time.Time
	lastEval     time.Time
}

// AlertServiceOptions 可选依赖：静默、处理人、内置规则评估。
type AlertServiceOptions struct {
	SilenceSvc  *AlertSilenceService
	AssigneeSvc *AlertRuleAssigneeService
	DutySvc     *AlertDutyService
}

type promEnrichTask struct {
	Fingerprint  string
	GeneratorURL string
}

// NewAlertService 创建相关逻辑。
func NewAlertService(db *gorm.DB, redisClient *redis.Client, sender mailer.Sender, cfg config.AlertConfig, opts *AlertServiceOptions) *AlertService {
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
	svc := &AlertService{
		db:               db,
		redis:            redisClient,
		mailer:           sender,
		cfg:              cfg,
		policySvc:        NewAlertPolicyService(db),
		monitorEvalState: make(map[uint]*monitorEvalRuleState),
		cloudExpiryState: make(map[string]bool),
	}
	if opts != nil {
		svc.silenceSvc = opts.SilenceSvc
		svc.assigneeSvc = opts.AssigneeSvc
		svc.dutySvc = opts.DutySvc
	}
	svc.startPrometheusEnrichWorkers()
	evalCtx, cancel := context.WithCancel(context.Background())
	svc.monitorEvalCancel = cancel
	go svc.runMonitorRuleEvaluator(evalCtx)
	return svc
}

func (s *AlertService) channelIDSetForAlert(ctx context.Context, status string, labels map[string]string) (map[uint]struct{}, string, string) {
	enabled, err := s.policySvc.ListEnabled(ctx)
	if err != nil || len(enabled) == 0 {
		return nil, "", ""
	}
	out := map[uint]struct{}{}
	matchedIDs := make([]string, 0)
	matchedNames := make([]string, 0)
	for _, p := range enabled {
		if status == "resolved" && !p.NotifyResolved {
			continue
		}
		ids := s.policySvc.MatchPolicyChannels(p, labels)
		if len(ids) > 0 {
			matchedIDs = append(matchedIDs, fmt.Sprintf("%d", p.ID))
			matchedNames = append(matchedNames, p.Name)
		}
		for _, id := range ids {
			out[id] = struct{}{}
		}
	}
	return out, strings.Join(matchedIDs, ","), strings.Join(matchedNames, ",")
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

type AlertChannelPreviewRequest struct {
	Settings map[string]interface{} `json:"settings"`
	Payload  map[string]interface{} `json:"payload"`
	Firing   bool                   `json:"firing"`
}

// PreviewChannelTemplate 渲染并返回通道模板预览文本。
func (s *AlertService) PreviewChannelTemplate(ctx context.Context, req AlertChannelPreviewRequest) (map[string]string, error) {
	settings := req.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	status := "resolved"
	if req.Firing {
		status = "firing"
	}
	// 使用统一渲染函数生成消息文本，title/severity 可从 payload 或设置中扩展。
	msg := s.renderChannelMessage(ctx, "Preview", "", status, payload, settings)
	return map[string]string{"message": msg}, nil
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

// ListChannels 查询列表相关的业务逻辑。
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

// CreateChannel 创建相关的业务逻辑。
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
		return nil, apperror.BadRequest("非邮件通道必须填写 Webhook 地址")
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

// UpdateChannel 更新相关的业务逻辑。
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
		return nil, apperror.BadRequest("非邮件通道必须填写 Webhook 地址")
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

// DeleteChannel 删除相关的业务逻辑。
func (s *AlertService) DeleteChannel(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.AlertChannel{}, id).Error
}

// TestChannel 测试相关的业务逻辑。
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
		content = "这是一条来自 yunshu 的测试告警消息。"
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

// ListEvents 查询列表相关的业务逻辑。
func (s *AlertService) ListEvents(ctx context.Context, q AlertEventListQuery) (list []model.AlertEvent, total int64, page int, pageSize int, err error) {
	page, pageSize = pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertEvent{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("title LIKE ? OR error_message LIKE ? OR channel_name LIKE ?", like, like, like)
	}
	if v := strings.TrimSpace(q.Cluster); v != "" {
		tx = tx.Where("cluster = ?", v)
	}
	if v := strings.TrimSpace(q.MonitorPipeline); v != "" {
		tx = tx.Where("monitor_pipeline = ?", v)
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

func parseLabelUint(v string) (uint, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func mergeNotifyEmailsUnique(emails []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func (s *AlertService) enrichAssigneeAndDutyEmails(ctx context.Context, outgoing map[string]interface{}, labels map[string]string) {
	rid, ok := parseLabelUint(labels["monitor_rule_id"])
	if !ok {
		return
	}
	var emails []string
	if s.assigneeSvc != nil {
		e, _ := s.assigneeSvc.ResolveNotifyEmails(ctx, rid)
		emails = append(emails, e...)
	}
	if s.dutySvc != nil {
		e, _ := s.dutySvc.ResolveNotifyEmailsAtRule(ctx, rid, time.Now())
		emails = append(emails, e...)
	}
	emails = mergeNotifyEmailsUnique(emails)
	if len(emails) > 0 {
		outgoing["assignee_emails"] = emails
	}
}

func (s *AlertService) logSilenceSuppressed(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, silenceID uint, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          "alertmanager",
		Title:           title + " (silence suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: monitorPipelineFromPayload(payload),
		GroupKey:        groupKey,
		LabelsDigest:    labelsDigest,
		ChannelID:       0,
		ChannelName:     "（静默抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    fmt.Sprintf("silence_id=%d", silenceID),
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText("suppressed by platform silence", s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) logNoMatchedChannel(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, payload map[string]interface{}, reason string) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:             "alertmanager",
		Title:              title + " (no matched channel)",
		Severity:           severity,
		Status:             status,
		Cluster:            cluster,
		MonitorPipeline:    monitorPipelineFromPayload(payload),
		GroupKey:           groupKey,
		LabelsDigest:       labelsDigest,
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matched_policy_ids"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matched_policy_names"),
		ChannelID:          0,
		ChannelName:        "（无匹配通道）",
		Success:            false,
		HTTPStatusCode:     0,
		ErrorMessage:       reason,
		RequestPayload:     truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload:    "",
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
}

// ReceiveAlertmanager 执行对应的业务逻辑。
func (s *AlertService) ReceiveAlertmanager(ctx context.Context, payload AlertManagerPayload) error {
	var channels []model.AlertChannel
	if err := s.db.WithContext(ctx).Model(&model.AlertChannel{}).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&channels).Error; err != nil {
		return err
	}
	for _, alert := range payload.Alerts {
		labels := mergeStringMap(payload.CommonLabels, alert.Labels)
		monitorPipeline := monitorPipelinePrometheus
		if s.isPlatformMonitor(labels, payload.Receiver) {
			monitorPipeline = monitorPipelinePlatform
		}
		labels["monitor_pipeline"] = monitorPipeline
		annotations := mergeStringMap(payload.CommonAnnotations, alert.Annotations)
		status := strings.TrimSpace(alert.Status)
		if status == "" {
			status = strings.TrimSpace(payload.Status)
		}
		status = strings.ToLower(strings.TrimSpace(status))
		if status == "" {
			status = "firing"
		}
		eventName := strings.TrimSpace(labels["alertname"])
		if eventName == "" {
			eventName = strings.TrimSpace(payload.CommonLabels["alertname"])
		}
		if eventName == "" {
			eventName = "Alertmanager 告警"
		}
		summary := strings.TrimSpace(annotations["summary"])
		if summary == "" {
			summary = strings.TrimSpace(annotations["description"])
		}
		if summary == "" {
			summary = strings.TrimSpace(payload.CommonAnnotations["summary"])
		}
		if summary == "" {
			summary = "Alertmanager webhook message"
		}
		severity := strings.TrimSpace(labels["severity"])
		if severity == "" {
			severity = strings.TrimSpace(payload.CommonLabels["severity"])
		}
		if severity == "" {
			severity = "warning"
		}
		title := eventName

		dims := alertnotify.ExtractDims(labels)
		groupKey := s.computeGroupKey(payload.Receiver, status, severity, eventName, labels, dims)
		labelsDigest := alertnotify.DigestLabels(labels)
		envLabel := s.resolveAlertEnvironmentLabel(labels, payload.Receiver, dims, alert.Labels)
		if s.silenceSvc != nil {
			if sid, muted, err := s.silenceSvc.FirstMatchingSilenceID(ctx, labels, time.Now()); err == nil && muted {
				minPayload := map[string]interface{}{
					"labels": labels, "annotations": annotations, "severity": severity, "status": status,
					"receiver": payload.Receiver, "fingerprint": alert.Fingerprint,
					"group_key": groupKey, "cluster": envLabel, "labels_digest": labelsDigest,
					"monitor_pipeline": monitorPipeline,
				}
				s.logSilenceSuppressed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, sid, minPayload)
				continue
			}
		}

		count, deduped, _ := s.updateFingerprintState(ctx, alert.Fingerprint, status)

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
		outgoing := map[string]interface{}{
			"source":           "alertmanager",
			"title":            title,
			"summary":          summary,
			"severity":         severity,
			"status":           status,
			"receiver":         payload.Receiver,
			"fingerprint":      alert.Fingerprint,
			"count":            count,
			"labels":           labels,
			"annotations":      annotations,
			"group_labels":     payload.GroupLabels,
			"am_version":       payload.Version,
			"starts_at":        alert.StartsAt,
			"ends_at":          alert.EndsAt,
			"generator_url":    alert.GeneratorURL,
			"current":          currentValue,
			"truncated":        payload.TruncatedAlerts,
			"occurred_at":      time.Now().Format(time.RFC3339),
			"cluster":          envLabel,
			"monitor_pipeline": monitorPipeline,
			"group_key":        groupKey,
			"labels_digest":    labelsDigest,
		}
		// Prometheus/Alertmanager 规则可在 labels 中携带 project_id，此处查库写入 project_name，供统一标题与通道展示。
		s.enrichOutgoingProjectName(ctx, outgoing)
		s.enrichAssigneeAndDutyEmails(ctx, outgoing, labels)

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
		policyChannels, matchedPolicyIDs, matchedPolicyNames := s.channelIDSetForAlert(ctx, status, labels)
		outgoing["matched_policy_ids"] = matchedPolicyIDs
		outgoing["matched_policy_names"] = matchedPolicyNames
		sentCount := 0
		for i := range channels {
			if len(policyChannels) > 0 {
				if _, ok := policyChannels[channels[i].ID]; !ok {
					continue
				}
			}
			settings, _ := parseChannelSettings(channels[i].HeadersJSON)
			if !channelMatchesAlert(settings, labels, dims) {
				continue
			}
			sentCount++
			_, _, _ = s.sendToChannel(ctx, &channels[i], "alertmanager", title, severity, status, outgoing)
		}
		if sentCount == 0 {
			reason := "no enabled channel matched"
			if len(channels) == 0 {
				reason = "no enabled channels"
			} else if len(policyChannels) > 0 {
				reason = "no channel matched policy"
			}
			s.logNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, reason)
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

// ValidateWebhookToken 校验相关的业务逻辑。
func (s *AlertService) ValidateWebhookToken(token string) bool {
	expected := strings.TrimSpace(s.cfg.WebhookToken)
	if expected == "" {
		return true
	}
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token) == expected
}

func (s *AlertService) isPlatformMonitor(labels map[string]string, receiver string) bool {
	if strings.TrimSpace(receiver) == "platform-monitor" {
		return true
	}
	if labels != nil && strings.TrimSpace(labels["source"]) == "prometheus_monitor" {
		return true
	}
	return false
}

func (s *AlertService) resolveAlertEnvironmentLabel(labels map[string]string, receiver string, dims alertnotify.Dims, alertLabels map[string]string) string {
	if s.isPlatformMonitor(labels, receiver) {
		// 集群列仅承载「环境/cluster」语义；平台规则未同步 cluster 标签时不要写死 unknown，回退到实例/节点等展示维度。
		if v := strings.TrimSpace(labels["cluster"]); v != "" {
			return v
		}
		return alertnotify.InferEnvironmentDisplay(labels, dims)
	}
	envLabel := strings.TrimSpace(dims.Cluster)
	if envLabel == "" {
		envLabel = alertnotify.InferEnvironmentDisplay(alertLabels, dims)
	}
	return envLabel
}

func monitorPipelineFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", payload["monitor_pipeline"]))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func mergeStringMap(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

type AlertHistoryStats struct {
	Total                 int64    `json:"total"`
	Firing                int64    `json:"firing"`
	Resolved              int64    `json:"resolved"`
	Success               int64    `json:"success"`
	Failed                int64    `json:"failed"`
	TodayCreated          int64    `json:"today_created"`
	ClusterValues         []string `json:"cluster_values"`
	MonitorPipelineValues []string `json:"monitor_pipeline_values"`
}

func (s *AlertService) HistoryStats(ctx context.Context) (*AlertHistoryStats, error) {
	stats := &AlertHistoryStats{}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("LOWER(TRIM(status)) = ?", "firing").Count(&stats.Firing).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("LOWER(TRIM(status)) = ?", "resolved").Count(&stats.Resolved).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("success = ?", true).Count(&stats.Success).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("success = ?", false).Count(&stats.Failed).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("DATE(created_at) = CURRENT_DATE").Count(&stats.TodayCreated).Error; err != nil {
		return nil, err
	}
	var clusters []string
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(cluster, '')) != ''").
		Group("cluster").
		Order("cluster ASC").
		Limit(500).
		Pluck("cluster", &clusters).Error; err != nil {
		return nil, err
	}
	stats.ClusterValues = clusters
	var pipes []string
	if err := s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(monitor_pipeline, '')) != ''").
		Group("monitor_pipeline").
		Order("monitor_pipeline ASC").
		Limit(32).
		Pluck("monitor_pipeline", &pipes).Error; err != nil {
		return nil, err
	}
	stats.MonitorPipelineValues = pipes
	return stats, nil
}
