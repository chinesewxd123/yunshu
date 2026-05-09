package service

import (
	"context"
	"encoding/json"
	"time"
	"yunshu/internal/pkg/constants"

	"github.com/redis/go-redis/v9"
)

const redisKeyAlertWebhookQueue = "alert:webhook:queue"

func (s *AlertService) shouldEnqueueAlertmanagerWebhook() bool {
	return s.redis != nil && !s.cfg.WebhookAsyncDisabled
}

func (s *AlertService) enqueueAlertmanagerWebhook(ctx context.Context, payload AlertManagerPayload) error {
	bs, err := json.Marshal(payload)
	if err != nil {
		return constants.ErrInternalWithMsg(constants.ErrMsg39d72e4b8516)
	}
	maxLen := s.cfg.WebhookQueueMaxLen
	if maxLen <= 0 {
		maxLen = 10000
	}
	n, err := s.redis.LLen(ctx, redisKeyAlertWebhookQueue).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if n >= int64(maxLen) {
		return constants.ErrInternalWithMsg(constants.ErrMsgfd7c760c8d45)
	}
	if err := s.redis.RPush(ctx, redisKeyAlertWebhookQueue, bs).Err(); err != nil {
		return err
	}
	return nil
}

func (s *AlertService) runAlertWebhookIngestWorker(ctx context.Context) {
	if s.redis == nil || s.cfg.WebhookAsyncDisabled {
		return
	}
	go func() {
		for {
			res, err := s.redis.BRPop(ctx, 5*time.Second, redisKeyAlertWebhookQueue).Result()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if err == redis.Nil {
					continue
				}
				continue
			}
			if len(res) < 2 {
				continue
			}
			var payload AlertManagerPayload
			if json.Unmarshal([]byte(res[1]), &payload) != nil {
				continue
			}
			procCtx := context.Background()
			_ = s.receiveAlertmanagerPayloadSync(procCtx, payload)
		}
	}()
}
