package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type MysqlBackupRepository struct {
	db *gorm.DB
}

func NewMysqlBackupRepository(db *gorm.DB) *MysqlBackupRepository {
	return &MysqlBackupRepository{db: db}
}

func (r *MysqlBackupRepository) CreateInstance(ctx context.Context, inst *model.MysqlBackupInstance) error {
	return r.db.WithContext(ctx).Create(inst).Error
}

func (r *MysqlBackupRepository) UpdateInstance(ctx context.Context, inst *model.MysqlBackupInstance) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

func (r *MysqlBackupRepository) DeleteInstance(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.MysqlBackupInstance{}, id).Error
}

func (r *MysqlBackupRepository) GetInstance(ctx context.Context, id uint) (*model.MysqlBackupInstance, error) {
	var inst model.MysqlBackupInstance
	err := r.db.WithContext(ctx).First(&inst, id).Error
	return &inst, err
}

func (r *MysqlBackupRepository) GetInstanceInProject(ctx context.Context, projectID, id uint) (*model.MysqlBackupInstance, error) {
	var inst model.MysqlBackupInstance
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&inst, id).Error
	return &inst, err
}

type MysqlBackupInstanceListParams struct {
	ProjectID uint
	Page      int
	PageSize  int
}

func (r *MysqlBackupRepository) ListInstances(ctx context.Context, p MysqlBackupInstanceListParams) ([]model.MysqlBackupInstance, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.MysqlBackupInstance{}).Where("project_id = ?", p.ProjectID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.MysqlBackupInstance
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *MysqlBackupRepository) CreateJob(ctx context.Context, job *model.MysqlBackupJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *MysqlBackupRepository) UpdateJob(ctx context.Context, job *model.MysqlBackupJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *MysqlBackupRepository) GetJob(ctx context.Context, id uint) (*model.MysqlBackupJob, error) {
	var job model.MysqlBackupJob
	err := r.db.WithContext(ctx).First(&job, id).Error
	return &job, err
}

type MysqlBackupJobListParams struct {
	ProjectID  uint
	InstanceID uint
	Page       int
	PageSize   int
}

func (r *MysqlBackupRepository) ListScheduleEnabledInstances(ctx context.Context) ([]model.MysqlBackupInstance, error) {
	var list []model.MysqlBackupInstance
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND schedule_enabled = ?", true, true).
		Where("cron_spec <> ''").
		Find(&list).Error
	return list, err
}

func (r *MysqlBackupRepository) TouchLastScheduledAt(ctx context.Context, id uint, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.MysqlBackupInstance{}).
		Where("id = ?", id).
		Update("last_scheduled_at", at).Error
}

func (r *MysqlBackupRepository) HasRunningJob(ctx context.Context, instanceID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.MysqlBackupJob{}).
		Where("instance_id = ? AND status = ?", instanceID, "running").
		Count(&n).Error
	return n > 0, err
}

func (r *MysqlBackupRepository) ListJobs(ctx context.Context, p MysqlBackupJobListParams) ([]model.MysqlBackupJob, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.MysqlBackupJob{}).Where("project_id = ?", p.ProjectID)
	if p.InstanceID > 0 {
		q = q.Where("instance_id = ?", p.InstanceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.MysqlBackupJob
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
