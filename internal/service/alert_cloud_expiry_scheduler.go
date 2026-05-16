package service

import (
	"context"

	"github.com/robfig/cron/v3"
)

// cloudExpirySchedulerSpec 云到期内置轮询节拍（六段式，含秒），仅用于判断是否到达各规则在控制台配置的 Cron；
// 与 alert.monitor_eval_cron_spec（内置 PromQL 监控规则）解耦。
const cloudExpirySchedulerSpec = "*/5 * * * * *"

func (s *AlertService) runCloudExpiryEvaluator(ctx context.Context) {
	spec := cloudExpirySchedulerSpec
	c := cron.New(cron.WithSeconds())
	job := func() {
		if ctx.Err() != nil {
			return
		}
		if s.infoLog != nil {
			s.infoLog.Info("cloud_expiry_scheduler wake", "inner_cron", spec)
		}
		_ = s.tickCloudExpiryRules(ctx)
	}
	if _, err := c.AddFunc(spec, job); err != nil {
		if s.infoLog != nil {
			s.infoLog.Error("cloud_expiry_scheduler init failed", "spec", spec, "err", err)
		}
		return
	}
	c.Start()
	<-ctx.Done()
	stopCtx := c.Stop()
	<-stopCtx.Done()
}
