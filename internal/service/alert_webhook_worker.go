package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
		return svcerr.InternalMsg("alert.webhook", "api", constants.ErrMsg39d72e4b8516)
	}
	maxLen := s.cfg.WebhookQueueMaxLen
	if maxLen <= 0 {
		maxLen = 10000
	}
	ok, err := luaEnqueueAlertWebhook.Run(ctx, s.redis, []string{redisKeyAlertWebhookQueue}, maxLen, string(bs)).Int()
	if err != nil {
		return svcerr.Pass("alert.webhook", "enqueueAlertmanagerWebhook", err)
	}
	if ok == 0 {
		return svcerr.InternalMsg("alert.webhook", "api", constants.ErrMsgfd7c760c8d45)
	}
	return nil
}

func (s *AlertService) logWebhook(level string, msg string, attrs ...any) {
	if s.infoLog == nil {
		return
	}
	switch level {
	case "warn":
		s.infoLog.Warn(msg, attrs...)
	case "error":
		s.infoLog.Error(msg, attrs...)
	default:
		s.infoLog.Info(msg, attrs...)
	}
}

func webhookPayloadLogAttrs(payload AlertManagerPayload) []any {
	attrs := []any{
		slog.String("status", payload.Status),
		slog.String("receiver", payload.Receiver),
		slog.Int("alerts", len(payload.Alerts)),
	}
	if len(payload.Alerts) > 0 {
		labels := payload.Alerts[0].Labels
		if labels != nil {
			if v := labels["alertname"]; v != "" {
				attrs = append(attrs, slog.String("alertname", v))
			}
		}
	}
	return attrs
}

func (s *AlertService) ingestWebhookPayloadWithRetry(ctx context.Context, payload AlertManagerPayload) {
	var lastErr error
	for attempt := 1; attempt <= alertWebhookMaxAttempts; attempt++ {
		lastErr = s.receiveAlertmanagerPayloadSync(ctx, payload)
		if lastErr == nil {
			if attempt > 1 {
				s.logWebhook("info", "alert webhook ingest succeeded after retry",
					append(webhookPayloadLogAttrs(payload), slog.Int("attempt", attempt))...)
			}
			return
		}
		s.logWebhook("warn", "alert webhook ingest failed",
			append(webhookPayloadLogAttrs(payload), slog.Int("attempt", attempt), slog.Any("error", lastErr))...)
		if attempt < alertWebhookMaxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	s.logWebhook("error", "alert webhook ingest exhausted retries",
		append(webhookPayloadLogAttrs(payload), slog.Any("error", lastErr))...)
}

func (s *AlertService) runAlertWebhookIngestWorker(ctx context.Context) {
	if s.redis == nil || s.cfg.WebhookAsyncDisabled {
		return
	}
	s.logWebhook("info", "alert webhook async worker started")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logWebhook("error", "alert webhook worker panic", slog.Any("error", r))
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
				s.logWebhook("warn", "alert webhook queue BRPop failed", slog.Any("error", err))
				time.Sleep(time.Second)
				continue
			}
			if len(res) < 2 {
				continue
			}
			raw := res[1]
			var payload AlertManagerPayload
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				s.logWebhook("warn", "alert webhook queue payload unmarshal failed",
					slog.Any("error", err), slog.Int("bytes", len(raw)))
				continue
			}
			procCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			s.ingestWebhookPayloadWithRetry(procCtx, payload)
			cancel()
		}
	}()
}
