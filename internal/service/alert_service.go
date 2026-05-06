package service

import (
	"context"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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
	AlertIP         string `form:"alertIP"`
	Status          string `form:"status"`
	MonitorPipeline string `form:"monitorPipeline"`
	GroupKey        string `form:"groupKey"`
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

	silenceSvc  *AlertSilenceService
	assigneeSvc *AlertRuleAssigneeService
	dutySvc     *AlertDutyService

	monitorEvalCancel context.CancelFunc
	monitorEvalMu     sync.Mutex
	aead              cipher.AEAD
	cloudExpiryState  map[string]bool

	// 可选依赖：告警抑制、订阅树路由
	inhibitionSvc   *AlertInhibitionService   // 告警抑制服务
	subscriptionSvc *AlertSubscriptionService // 订阅树服务
	receiverGroupCache *ReceiverGroupCache    // 接收组缓存

	metrics        *AlertMetrics // Prometheus自监控指标
	metricsUpdater *AlertMetricsUpdater
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
	if cfg.GroupWaitSeconds < 0 {
		cfg.GroupWaitSeconds = 0
	}
	if cfg.GroupIntervalSeconds <= 0 {
		cfg.GroupIntervalSeconds = 60
	}
	if cfg.RepeatIntervalSeconds <= 0 {
		cfg.RepeatIntervalSeconds = 300
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
		cloudExpiryState: make(map[string]bool),
		inhibitionSvc:    NewAlertInhibitionService(db, redisClient),
		subscriptionSvc:  NewAlertSubscriptionService(db),
		receiverGroupCache: NewReceiverGroupCache(db),
		metrics:          NewAlertMetrics(),
	}

	// 初始化指标更新器并启动
	svc.metricsUpdater = NewAlertMetricsUpdater(svc.metrics, svc.inhibitionSvc)
	svc.metricsUpdater.Start()

	if opts != nil {
		svc.silenceSvc = opts.SilenceSvc
		svc.assigneeSvc = opts.AssigneeSvc
		svc.dutySvc = opts.DutySvc
	}
	svc.startPrometheusEnrichWorkers()
	svc.startInhibitionPruner(context.Background())
	evalCtx, cancel := context.WithCancel(context.Background())
	svc.monitorEvalCancel = cancel
	go svc.runMonitorRuleEvaluator(evalCtx)
	return svc
}

func (s *AlertService) GetSubscriptionService() *AlertSubscriptionService {
	return s.subscriptionSvc
}

func (s *AlertService) channelIDSetForAlert(ctx context.Context, status string, labels map[string]string) (map[uint]struct{}, string, string, int) {
	// 彻底弃用旧策略：仅使用订阅树路由（订阅节点 -> 接收组 -> 通道）
	if s.subscriptionSvc == nil || s.receiverGroupCache == nil {
		return nil, "", "", 0
	}
	projectID := parseLabelUintOrZero(labels["project_id"])
	severity := strings.TrimSpace(labels["severity"])
	route, ok := s.subscriptionSvc.MatchRouteDetailed(ctx, projectID, labels, severity, status)
	if !ok || len(route.ReceiverGroupIDs) == 0 {
		return nil, "", "", 0
	}
	out := map[uint]struct{}{}
	for _, gid := range route.ReceiverGroupIDs {
		g, err := s.receiverGroupCache.Get(gid)
		if err != nil || g == nil {
			continue
		}
		if !g.IsActiveNow() {
			continue
		}
		for _, cid := range g.ChannelIDs {
			if cid > 0 {
				out[cid] = struct{}{}
			}
		}
	}
	ids := make([]string, 0, len(route.MatchedNodeIDs))
	for _, id := range route.MatchedNodeIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	return out, strings.Join(ids, ","), strings.Join(route.MatchedNodeNames, ","), route.SilenceSeconds
}

func parseLabelUintOrZero(s string) uint {
	n, ok := parseLabelUint(s)
	if !ok {
		return 0
	}
	return n
}

func (s *AlertService) shouldSuppressByRouteSilence(ctx context.Context, status, groupKey, matchedNodeIDs string, silenceSeconds int, labels map[string]string) bool {
	if s.redis == nil || silenceSeconds <= 0 || status != "firing" {
		return false
	}
	gk := strings.TrimSpace(groupKey)
	nid := strings.TrimSpace(matchedNodeIDs)
	if gk == "" || nid == "" {
		return false
	}
	key := "alert:subscription:silence:" + gk + ":" + nid
	ok, err := s.redis.SetNX(ctx, key, "1", time.Duration(silenceSeconds)*time.Second).Result()
	if err != nil {
		return false
	}
	if labels != nil {
		if ruleID, parsed := parseLabelUint(labels["monitor_rule_id"]); parsed && ruleID > 0 {
			// 为规则列表页提供可观测状态：订阅静默窗口剩余时间
			_ = s.redis.Set(ctx, fmt.Sprintf("alert:subscription:silence:rule:%d", ruleID), nid, time.Duration(silenceSeconds)*time.Second).Err()
		}
	}
	return !ok
}

func (s *AlertService) logSuppressedRouteSilence(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, silenceSeconds int, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          "alertmanager",
		Title:           title + " (subscription silence suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(groupKey),
		LabelsDigest:    strings.TrimSpace(labelsDigest),
		ChannelName:     "（未外发·订阅静默窗口抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    "subscription_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by subscription silence_seconds=%d", silenceSeconds), s.cfg.MaxPayloadChars),
	}
	_ = s.db.WithContext(ctx).Create(&event).Error
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
	Settings         map[string]interface{} `json:"settings"`
	Payload          map[string]interface{} `json:"payload"`
	Firing           bool                   `json:"firing"`
	TemplateFiring   string                 `json:"template_firing"`
	TemplateResolved string                 `json:"template_resolved"`
	Status           string                 `json:"status"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	Severity         string                 `json:"severity"`
	ProjectID        uint                   `json:"project_id"`
	RawPayloadJSON   string                 `json:"raw_payload_json"`
}

type AlertChannelPreviewResponse struct {
	Rendered           string                 `json:"rendered"`
	SamplePayload      map[string]interface{} `json:"sample_payload"`
	AvailableFields    []string               `json:"available_fields"`
	RawPayloadFields   []string               `json:"raw_payload_fields"`
	CombinedFields     []string               `json:"combined_fields"`
	SuggestedLabelKeys []string               `json:"suggested_label_keys"`
}

// PreviewChannelTemplate 渲染并返回通道模板预览文本。
func (s *AlertService) PreviewChannelTemplate(ctx context.Context, req AlertChannelPreviewRequest) (*AlertChannelPreviewResponse, error) {
	settings := req.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if raw := strings.TrimSpace(req.RawPayloadJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &payload)
	}
	if labelsAny, ok := payload["labels"]; !ok || labelsAny == nil {
		payload["labels"] = map[string]interface{}{}
	}
	if labels, ok := payload["labels"].(map[string]interface{}); ok {
		if req.ProjectID > 0 {
			labels["project_id"] = req.ProjectID
		}
		if strings.TrimSpace(fmt.Sprintf("%v", labels["alertname"])) == "" {
			labels["alertname"] = "PreviewAlert"
		}
	}
	if strings.TrimSpace(req.TemplateFiring) != "" {
		settings["messageTemplateFiring"] = strings.TrimSpace(req.TemplateFiring)
	}
	if strings.TrimSpace(req.TemplateResolved) != "" {
		settings["messageTemplateResolved"] = strings.TrimSpace(req.TemplateResolved)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", payload["summary"])) == "" && strings.TrimSpace(req.Content) != "" {
		payload["summary"] = strings.TrimSpace(req.Content)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", payload["severity"])) == "" && strings.TrimSpace(req.Severity) != "" {
		payload["severity"] = strings.TrimSpace(req.Severity)
	}
	if strings.TrimSpace(fmt.Sprintf("%v", payload["status"])) == "" && strings.TrimSpace(req.Status) != "" {
		payload["status"] = strings.TrimSpace(req.Status)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" && req.Firing {
		status = "firing"
	}
	if status == "" {
		status = "resolved"
	}
	// 使用统一渲染函数生成消息文本，title/severity 可从 payload 或设置中扩展。
	msg := s.renderChannelMessage(ctx, alertnotify.SafeOr(strings.TrimSpace(req.Title), "Preview"), strings.TrimSpace(req.Severity), status, payload, settings)
	rawFields, combinedFields, labelKeys := previewPayloadFieldCatalog(payload)
	return &AlertChannelPreviewResponse{
		Rendered:           msg,
		SamplePayload:      payload,
		AvailableFields:    []string{"Title", "Severity", "Status", "StatusText", "Summary", "Description", "ProjectName", "Cluster", "OccurredAt", "StartsAt", "EndsAt", "Current", "Count", "Fingerprint", "GeneratorURL", "Labels", "LabelsText"},
		RawPayloadFields:   rawFields,
		CombinedFields:     combinedFields,
		SuggestedLabelKeys: labelKeys,
	}, nil
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
		"source":     "manual-test",
		"title":      title,
		"content":    content,
		"summary":    content,
		"severity":   severity,
		"status":     "firing",
		"occurredAt": time.Now().Format(time.RFC3339),
		"cluster":    "manual-test",
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
	if v := strings.TrimSpace(q.AlertIP); v != "" {
		like := "%" + v + "%"
		tx = tx.Where(
			"cluster = ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ?",
			v,
			"%\"instance\":\""+v+"\"%",
			"%\"pod_ip\":\""+v+"\"%",
			"%\"node\":\""+v+"\"%",
			"%\"ip\":\""+v+"\"%",
			like,
		)
	}
	if v := strings.ToLower(strings.TrimSpace(q.Status)); v != "" {
		tx = tx.Where("LOWER(TRIM(status)) = ?", v)
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
	for i := range list {
		hydrateAlertEvent(&list[i])
	}
	s.backfillResolvedAlertIP(ctx, list)
	return list, total, page, pageSize, nil
}

func (s *AlertService) backfillResolvedAlertIP(ctx context.Context, list []model.AlertEvent) {
	if len(list) == 0 {
		return
	}
	missing := map[string]struct{}{}
	for i := range list {
		st := strings.ToLower(strings.TrimSpace(list[i].Status))
		if st != "resolved" {
			continue
		}
		if strings.TrimSpace(list[i].AlertIP) != "" {
			continue
		}
		gk := strings.TrimSpace(list[i].GroupKey)
		if gk == "" {
			continue
		}
		missing[gk] = struct{}{}
	}
	if len(missing) == 0 {
		return
	}
	groupKeys := make([]string, 0, len(missing))
	for k := range missing {
		groupKeys = append(groupKeys, k)
	}
	var firingRows []model.AlertEvent
	if err := s.db.WithContext(ctx).
		Where("group_key IN ? AND LOWER(TRIM(status)) = ?", groupKeys, "firing").
		Order("id DESC").
		Find(&firingRows).Error; err != nil {
		return
	}
	ipByGroup := map[string]string{}
	for i := range firingRows {
		row := firingRows[i]
		gk := strings.TrimSpace(row.GroupKey)
		if gk == "" {
			continue
		}
		if _, ok := ipByGroup[gk]; ok {
			continue
		}
		hydrateAlertEvent(&row)
		ip := strings.TrimSpace(row.AlertIP)
		if ip != "" {
			ipByGroup[gk] = ip
		}
	}
	for i := range list {
		if strings.ToLower(strings.TrimSpace(list[i].Status)) != "resolved" || strings.TrimSpace(list[i].AlertIP) != "" {
			continue
		}
		if ip := strings.TrimSpace(ipByGroup[strings.TrimSpace(list[i].GroupKey)]); ip != "" {
			list[i].AlertIP = ip
		}
	}
}

func extractAlertIPFromPayload(requestPayload, fallback string) string {
	labels, _ := extractAlertPayloadLabels(requestPayload)
	for _, key := range []string{"instance", "pod_ip", "ip", "node"} {
		v := strings.TrimSpace(labels[key])
		if v != "" {
			return v
		}
	}
	return strings.TrimSpace(fallback)
}

func extractAlertStartedAtFromPayload(requestPayload string) string {
	raw := strings.TrimSpace(requestPayload)
	if raw == "" {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	for _, key := range []string{"startsAt"} {
		v := strings.TrimSpace(fmt.Sprintf("%v", payload[key]))
		if v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func extractAlertPayloadLabels(requestPayload string) (map[string]string, map[string]interface{}) {
	raw := strings.TrimSpace(requestPayload)
	if raw != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			if labelsAny, ok := payload["labels"]; ok {
				if labels, ok := labelsAny.(map[string]interface{}); ok {
					out := make(map[string]string, len(labels))
					for key, value := range labels {
						v := strings.TrimSpace(fmt.Sprintf("%v", value))
						if v != "" && v != "<nil>" {
							out[key] = v
						}
					}
					return out, payload
				}
			}
			// 非 labels 结构（如钉钉/企微下发体）也返回 payload，
			// 便于从 atMobiles/mentioned_mobile_list 等字段提取接收人。
			return map[string]string{}, payload
		}
	}
	return map[string]string{}, nil
}

func parseUintCSV(raw string) []uint {
	var out []uint
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		n, ok := parseLabelUint(part)
		if ok && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

func parseTrimmedCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseStringListAny(v interface{}) []string {
	raw := normalizeRecipientList(v)
	var out []string
	for _, one := range raw {
		s := strings.TrimSpace(one)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func payloadValueByPath(payload map[string]interface{}, path string) interface{} {
	if payload == nil {
		return nil
	}
	cur := interface{}(payload)
	for _, part := range strings.Split(strings.TrimSpace(path), ".") {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		obj, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur, ok = obj[key]
		if !ok {
			return nil
		}
	}
	return cur
}

func uniqTrimmedStrings(in []string, lower bool) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, one := range in {
		s := strings.TrimSpace(one)
		if s == "" {
			continue
		}
		k := s
		if lower {
			k = strings.ToLower(s)
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	return out
}

func extractEventReceivers(requestPayload, channelName string) []string {
	_, payload := extractAlertPayloadLabels(requestPayload)
	if payload == nil {
		return nil
	}
	ch := strings.ToLower(strings.TrimSpace(channelName))
	var out []string
	if strings.Contains(ch, "email") || strings.Contains(ch, "邮件") {
		for _, key := range []string{"assignee_emails", "to", "recipients", "emails"} {
			out = append(out, parseStringListAny(payloadValueByPath(payload, key))...)
		}
		return uniqTrimmedStrings(out, true)
	}
	if strings.Contains(ch, "ding") || strings.Contains(ch, "钉") || strings.Contains(ch, "wecom") || strings.Contains(ch, "wechat") || strings.Contains(ch, "企微") || strings.Contains(ch, "企业微信") {
		for _, key := range []string{"at.atMobiles", "text.mentioned_mobile_list", "atMobiles", "mentioned_mobile_list"} {
			out = append(out, parseStringListAny(payloadValueByPath(payload, key))...)
		}
		return uniqTrimmedStrings(out, false)
	}
	return nil
}

func hydrateAlertEvent(it *model.AlertEvent) {
	if it == nil {
		return
	}
	labels, _ := extractAlertPayloadLabels(it.RequestPayload)
	it.Environment = strings.TrimSpace(it.Cluster)
	if rawCluster := strings.TrimSpace(labels["cluster"]); rawCluster != "" {
		it.Cluster = rawCluster
	}
	it.AlertIP = extractAlertIPFromPayload(it.RequestPayload, it.Environment)
	it.AlertStartedAt = extractAlertStartedAtFromPayload(it.RequestPayload)
	it.MatchedPolicyIDList = parseUintCSV(it.MatchedPolicyIDs)
	it.MatchedPolicyNameList = parseTrimmedCSV(it.MatchedPolicyNames)
	it.ReceiverList = extractEventReceivers(it.RequestPayload, it.ChannelName)
}

func previewPayloadFieldCatalog(payload map[string]interface{}) ([]string, []string, []string) {
	rawFields := make([]string, 0)
	labelKeys := make([]string, 0)
	for key := range payload {
		rawFields = append(rawFields, key)
	}
	sort.Strings(rawFields)
	if labelsAny, ok := payload["labels"]; ok {
		if labels, ok := labelsAny.(map[string]interface{}); ok {
			for key := range labels {
				labelKeys = append(labelKeys, key)
			}
		}
	}
	sort.Strings(labelKeys)
	combined := append([]string{}, []string{"Title", "Severity", "Status", "StatusText", "Summary", "Description", "ProjectName", "Cluster", "OccurredAt", "StartsAt", "EndsAt", "Current", "Count", "Fingerprint", "GeneratorURL", "Labels", "LabelsText"}...)
	combined = append(combined, rawFields...)
	sort.Strings(combined)
	return rawFields, combined, labelKeys
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
		ErrorMessage:    "silence_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by platform silence_id=%d", silenceID), s.cfg.MaxPayloadChars),
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
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matchedPolicyNames"),
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
					"groupKey": groupKey, "cluster": envLabel, "labelsDigest": labelsDigest,
					"monitorPipeline": monitorPipeline,
				}
				s.logSilenceSuppressed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, sid, minPayload)
				continue
			}
		}

		count, _, _ := s.updateFingerprintState(ctx, alert.Fingerprint, status)

		// 构建outgoing payload（提前到抑制检查前）
		outgoing := map[string]interface{}{
			"source":          "alertmanager",
			"title":           title,
			"summary":         summary,
			"severity":        severity,
			"status":          status,
			"receiver":        payload.Receiver,
			"fingerprint":     alert.Fingerprint,
			"count":           count,
			"labels":          labels,
			"annotations":     annotations,
			"group_labels":    payload.GroupLabels,
			"am_version":      payload.Version,
			"startsAt":        alert.StartsAt,
			"endsAt":          alert.EndsAt,
			"generatorURL":    alert.GeneratorURL,
			"truncated":       payload.TruncatedAlerts,
			"occurredAt":      time.Now().Format(time.RFC3339),
			"cluster":         envLabel,
			"monitorPipeline": monitorPipeline,
			"groupKey":        groupKey,
			"labelsDigest":    labelsDigest,
		}

		// 告警抑制检查
		if status == "firing" && s.inhibitionSvc != nil {
			if inhibited, inhEvent := s.CheckInhibition(ctx, labels); inhibited {
				s.logInhibitionEvent(ctx, title, severity, status, envLabel, groupKey, labelsDigest, inhEvent, outgoing)
				// 记录为被抑制的源告警（如果它本身匹配源告警规则）
				_ = s.RecordSourceInhibition(ctx, labels)
				continue
			}
			// 记录源告警（如果匹配）
			_ = s.RecordSourceInhibition(ctx, labels)
		}

		// 清除源告警记录（当告警恢复时）
		if status == "resolved" && s.inhibitionSvc != nil {
			_ = s.ClearSourceInhibition(ctx, labels)
		}

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
		outgoing["current"] = currentValue
		// P3：异步增强，不阻塞通知主链路（缓存 miss 时上面已尽力同步查询）
		if status == "firing" {
			s.enqueuePrometheusEnrich(promEnrichTask{
				Fingerprint:  alert.Fingerprint,
				GeneratorURL: alert.GeneratorURL,
			})
		}
		// P3：异步增强，不阻塞通知主链路（缓存 miss 时上面已尽力同步查询）
		if status == "firing" {
			s.enqueuePrometheusEnrich(promEnrichTask{
				Fingerprint:  alert.Fingerprint,
				GeneratorURL: alert.GeneratorURL,
			})
		}
		// Prometheus/Alertmanager 规则可在 labels 中携带 project_id，此处查库写入 project_name，供统一标题与通道展示。
		s.enrichOutgoingProjectName(ctx, outgoing)
		s.enrichAssigneeAndDutyEmails(ctx, outgoing, labels)

		// 服务端第二层收敛：对齐 Alertmanager/N9E 的 group_wait/group_interval/repeat_interval
		if status == "firing" {
			// 新一轮 firing 开始前，清理“恢复已发送”标记，允许后续新的 resolved 再发送一次恢复通知。
			_ = s.clearResolvedNotificationSent(ctx, alert.Fingerprint)
			shouldSend, reason, aggCount, firstSeen, lastSeen := s.decideFiringGroupTiming(ctx, groupKey, labelsDigest)
			outgoing["agg_count"] = aggCount
			outgoing["agg_first_seen"] = firstSeen
			outgoing["agg_last_seen"] = lastSeen
			if !shouldSend {
				outgoing["suppressed_reason"] = reason
				s.logSuppressedFiringTiming(ctx, title, severity, status, groupKey, labelsDigest, reason, outgoing)
				continue
			}
		}
		if status == "resolved" {
			// 对齐夜莺/Alertmanager 语义：resolved 对同一 fingerprint 只发送一次。
			firstResolved, _ := s.markResolvedNotificationSent(ctx, alert.Fingerprint)
			if !firstResolved {
				outgoing["resolved_sent"] = false
				outgoing["summary"] = "重复恢复事件已抑制（同一告警实例仅发送一次恢复通知）。"
				s.logSuppressedResolvedAggregate(ctx, title, severity, status, groupKey, outgoing)
				continue
			}
			outgoing["resolved_sent"] = true
		}

		// firing 去重不应阻断 repeat interval 语义：
		// 具体“是否重复发送”由 groupKey 的 group_wait/group_interval/repeat_interval 状态机控制。
		subscriptionChannels, matchedPolicyIDs, matchedPolicyNames, subscriptionSilenceSeconds := s.channelIDSetForAlert(ctx, status, labels)
		outgoing["matchedPolicyIds"] = matchedPolicyIDs
		outgoing["matchedPolicyNames"] = matchedPolicyNames
		outgoing["subscription_silence_seconds"] = subscriptionSilenceSeconds
		if len(subscriptionChannels) == 0 {
			s.logNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, "no_policy_matched")
			continue
		}
		if s.shouldSuppressByRouteSilence(ctx, status, groupKey, matchedPolicyIDs, subscriptionSilenceSeconds, labels) {
			s.logSuppressedRouteSilence(ctx, title, severity, status, envLabel, groupKey, labelsDigest, subscriptionSilenceSeconds, outgoing)
			continue
		}
		sentCount := 0
		for i := range channels {
			if _, ok := subscriptionChannels[channels[i].ID]; !ok {
				continue
			}
			settings, _ := parseChannelSettings(channels[i].HeadersJSON)
			if !channelMatchesAlert(settings, labels, dims) {
				continue
			}
			sentCount++
			_, _, _ = s.sendToChannel(ctx, &channels[i], "alertmanager", title, severity, status, outgoing)
		}
		if sentCount == 0 {
			reason := "no_channel_matched"
			if len(channels) == 0 {
				reason = "no_enabled_channels"
			} else if len(subscriptionChannels) > 0 {
				reason = "no_channel_matched_subscription"
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
	s := strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"]))
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
