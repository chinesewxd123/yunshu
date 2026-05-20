package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"github.com/redis/go-redis/v9"
)

const redisKeyAlertWebhookQueue = "alert:webhook:queue"

// 原子入队：在队列长度未达上限时 RPush，避免 LLen+RPush 竞态突破 maxLen。
var luaEnqueueAlertWebhook = redis.NewScript(`
local n = redis.call('LLEN', KEYS[1])
if n >= tonumber(ARGV[1]) then
  return 0
end
redis.call('RPUSH', KEYS[1], ARGV[2])
return 1
`)

const (
	alertWebhookMaxAttempts = 3
	alertWebhookBRPopWait   = 5 * time.Second
)

func (s *AlertService) shouldEnqueueAlertmanagerWebhook() bool {
	return s.redis != nil && !s.cfg.WebhookAsyncDisabled
}

func (s *AlertService) enqueueAlertmanagerWebhook(ctx context.Context, payload AlertManagerPayload) error {
	bs, err := json.Marshal(payload)
	if err != nil {
		return svcerr.InternalMsg(ctx, "alert.webhook", "api", constants.ErrMsg39d72e4b8516)
	}
	maxLen := s.cfg.WebhookQueueMaxLen
	if maxLen <= 0 {
		maxLen = 10000
	}
	ok, err := luaEnqueueAlertWebhook.Run(ctx, s.redis, []string{redisKeyAlertWebhookQueue}, maxLen, string(bs)).Int()
	if err != nil {
		return svcerr.Pass(ctx, "alert.webhook", "enqueueAlertmanagerWebhook", err)
	}
	if ok == 0 {
		return svcerr.InternalMsg(ctx, "alert.webhook", "api", constants.ErrMsgfd7c760c8d45)
	}
	return nil
}

func webhookPayloadLogAttrs(payload AlertManagerPayload) []any {
	attrs := []any{
		"status", payload.Status,
		"receiver", payload.Receiver,
		"alerts", len(payload.Alerts),
	}
	if len(payload.Alerts) > 0 {
		labels := payload.Alerts[0].Labels
		if labels != nil {
			if v := labels["alertname"]; v != "" {
				attrs = append(attrs, "alertname", v)
			}
		}
	}
	return attrs
}

func (s *AlertService) logWebhookWarn(msg string, attrs ...any) {
	alertLog().Warnw(msg, attrs...)
}

func (s *AlertService) logWebhookError(err error, msg string, attrs ...any) {
	alertLog().Errorw(err, msg, attrs...)
}

func (s *AlertService) logWebhookInfo(msg string, attrs ...any) {
	alertLog().Infow(msg, attrs...)
}

func (s *AlertService) ingestWebhookPayloadWithRetry(ctx context.Context, payload AlertManagerPayload) {
	var lastErr error
	for attempt := 1; attempt <= alertWebhookMaxAttempts; attempt++ {
		lastErr = s.receiveAlertmanagerPayloadSync(ctx, payload)
		if lastErr == nil {
			if attempt > 1 {
				s.logWebhookInfo("Alert webhook ingest succeeded after retry",
					append(webhookPayloadLogAttrs(payload), "attempt", attempt)...)
			}
			return
		}
		s.logWebhookWarn("Failed to ingest alert webhook payload",
			append(webhookPayloadLogAttrs(payload), "attempt", attempt, "error", lastErr)...)
		if attempt < alertWebhookMaxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	s.logWebhookError(lastErr, "Alert webhook ingest exhausted retries", webhookPayloadLogAttrs(payload)...)
}

func (s *AlertService) runAlertWebhookIngestWorker(ctx context.Context) {
	if s.redis == nil || s.cfg.WebhookAsyncDisabled {
		return
	}
	s.logWebhookInfo("Started alert webhook async worker")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logWebhookError(errors.New("panic"), "Alert webhook worker panic", "panic", r)
			}
		}()
		for {
			if ctx.Err() != nil {
				return
			}
			res, err := s.redis.BRPop(ctx, alertWebhookBRPopWait, redisKeyAlertWebhookQueue).Result()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				s.logWebhookWarn("Alert webhook queue BRPop failed", "error", err)
				time.Sleep(time.Second)
				continue
			}
			if len(res) < 2 {
				continue
			}
			raw := res[1]
			var payload AlertManagerPayload
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				s.logWebhookWarn("Failed to unmarshal alert webhook queue payload", "error", err, "bytes", len(raw))
				continue
			}
			procCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			s.ingestWebhookPayloadWithRetry(procCtx, payload)
			cancel()
		}
	}()
}
