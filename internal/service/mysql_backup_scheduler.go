package service

import (
	"context"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
)

const defaultMysqlBackupInnerTick = "*/30 * * * * *"

var mysqlBackupCronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ValidateMysqlBackupCronSpec 校验实例 Cron 表达式（对齐云到期规则）。
func ValidateMysqlBackupCronSpec(spec string) error {
	return ValidateCloudExpiryCronSpec(spec)
}

func parseMysqlBackupCronSchedule(spec string) (cron.Schedule, error) {
	return mysqlBackupCronParser.Parse(strings.TrimSpace(spec))
}

func shouldRunMysqlBackupByCron(spec string, last *time.Time, now time.Time) bool {
	sched, err := parseMysqlBackupCronSchedule(spec)
	if err != nil {
		return false
	}
	if last == nil || last.IsZero() {
		return true
	}
	next := sched.Next(*last)
	return !now.Before(next)
}

// RunMysqlBackupScheduler 启动定时备份 Worker（字典 mysql_backup_scheduler_* 控制开关与节拍）。
func (s *MysqlBackupService) RunMysqlBackupScheduler(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	log := mysqlBackupLog()
	cfg := dictconfig.ResolveMysqlBackupSchedulerConfig(ctx, s.db, dictconfig.DefaultMysqlBackupSchedulerDictTypes())
	if !cfg.Enabled {
		log.Infow("MySQL backup scheduler disabled by dict")
		return
	}
	spec := strings.TrimSpace(cfg.TickSpec)
	if spec == "" {
		spec = defaultMysqlBackupInnerTick
	}
	c := cron.New(cron.WithSeconds())
	job := func() {
		if ctx.Err() != nil {
			return
		}
		if err := s.tickScheduledBackups(ctx); err != nil {
			log.Warnw("MySQL backup scheduler tick failed", "error", err)
		}
	}
	if _, err := c.AddFunc(spec, job); err != nil {
		log.Errorw(err, "Failed to init MySQL backup scheduler", "tick_spec", spec)
		return
	}
	log.Infow("Started MySQL backup scheduler", "tick_spec", spec)
	c.Start()
	<-ctx.Done()
	stopCtx := c.Stop()
	<-stopCtx.Done()
}

func (s *MysqlBackupService) tickScheduledBackups(ctx context.Context) error {
	if s.aead == nil {
		return nil
	}
	list, err := s.backupRepo.ListScheduleEnabledInstances(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range list {
		inst := &list[i]
		cronSpec := strings.TrimSpace(inst.CronSpec)
		if cronSpec == "" {
			continue
		}
		if !shouldRunMysqlBackupByCron(cronSpec, inst.LastScheduledAt, now) {
			continue
		}
		running, err := s.backupRepo.HasRunningJob(ctx, inst.ID)
		if err != nil {
			continue
		}
		if running {
			continue
		}
		s.runScheduledInstance(ctx, inst, now)
	}
	return nil
}

func (s *MysqlBackupService) runScheduledInstance(ctx context.Context, inst *model.MysqlBackupInstance, now time.Time) {
	s.schedMu.Lock()
	if s.schedRunning[inst.ID] {
		s.schedMu.Unlock()
		return
	}
	s.schedRunning[inst.ID] = true
	s.schedMu.Unlock()

	defer func() {
		s.schedMu.Lock()
		delete(s.schedRunning, inst.ID)
		s.schedMu.Unlock()
	}()

	_ = s.backupRepo.TouchLastScheduledAt(ctx, inst.ID, now)
	_, _ = s.enqueueBackup(ctx, inst.ProjectID, inst.ID, model.MysqlBackupTriggerScheduled)
}
