package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
)

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
	Rendered           string                                     `json:"rendered"`
	SamplePayload      map[string]interface{}                     `json:"sample_payload"`
	AvailableFields    []string                                   `json:"available_fields"`
	RawPayloadFields   []string                                   `json:"raw_payload_fields"`
	CombinedFields     []string                                   `json:"combined_fields"`
	SuggestedLabelKeys []string                                   `json:"suggested_label_keys"`
	TemplateVariables  []alertdispatch.ChannelTemplateVariableDoc `json:"template_variables"`
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
	docs := alertdispatch.ChannelTemplateVariableDocs()
	return &AlertChannelPreviewResponse{
		Rendered:           msg,
		SamplePayload:      payload,
		AvailableFields:    append([]string{}, alertdispatch.ChannelTemplateFieldList()...),
		RawPayloadFields:   rawFields,
		CombinedFields:     combinedFields,
		SuggestedLabelKeys: labelKeys,
		TemplateVariables:  docs,
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
		return nil, svcerr.Pass(ctx, "alert", "ListChannels", err)
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
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgaae2bd7c8c91)
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
		return nil, svcerr.Pass(ctx, "alert", "CreateChannel", err)
	}
	if err := validateEmailChannelRecipients(ch.Enabled, ch.Type, ch.HeadersJSON); err != nil {
		return nil, svcerr.Pass(ctx, "alert", "CreateChannel", err)
	}
	if err := s.db.WithContext(ctx).Create(ch).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "CreateChannel", err)
	}
	return ch, nil
}

// UpdateChannel 更新相关的业务逻辑。
func (s *AlertService) UpdateChannel(ctx context.Context, id uint, req AlertChannelUpsertRequest) (*model.AlertChannel, error) {
	var ch model.AlertChannel
	if err := s.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "UpdateChannel", err)
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
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgaae2bd7c8c91)
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
		return nil, svcerr.Pass(ctx, "alert", "UpdateChannel", err)
	}
	if err := validateEmailChannelRecipients(ch.Enabled, ch.Type, ch.HeadersJSON); err != nil {
		return nil, svcerr.Pass(ctx, "alert", "UpdateChannel", err)
	}
	if err := s.db.WithContext(ctx).Save(&ch).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert", "UpdateChannel", err)
	}
	return &ch, nil
}

// DeleteChannel 删除相关的业务逻辑。
func (s *AlertService) DeleteChannel(ctx context.Context, id uint) error {
	if err := s.db.WithContext(ctx).Delete(&model.AlertChannel{}, id).Error; err != nil {
		return svcerr.Pass(ctx, "alert", "DeleteChannel", err)
	}
	return nil
}

// TestChannel 测试相关的业务逻辑。
func (s *AlertService) TestChannel(ctx context.Context, id uint, req AlertTestRequest) error {
	var ch model.AlertChannel
	if err := s.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return svcerr.Pass(ctx, "alert", "TestChannel", err)
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
	_, _, err := s.sendToChannel(ctx, &ch, alertdispatch.NewEnvelope("manual-test", title, severity, "firing", payload))
	return svcerr.Pass(ctx, "alert", "TestChannel", err)
}

// ListEvents 查询列表相关的业务逻辑。
