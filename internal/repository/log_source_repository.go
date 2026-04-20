package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogSourceRepository struct{ db *gorm.DB }

func NewLogSourceRepository(db *gorm.DB) *LogSourceRepository { return &LogSourceRepository{db: db} }

type LogSourceListParams struct {
	ProjectID uint
	ServiceID *uint
	Page      int
	PageSize  int
}

func (r *LogSourceRepository) GetByID(ctx context.Context, id uint) (*model.ServiceLogSource, error) {
	var it model.ServiceLogSource
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogSourceRepository) GetByIDInProject(ctx context.Context, projectID uint, id uint) (*model.ServiceLogSource, error) {
	var it model.ServiceLogSource
	err := r.db.WithContext(ctx).
		Model(&model.ServiceLogSource{}).
		Joins("JOIN services ON services.id = service_log_sources.service_id").
		Joins("JOIN servers ON servers.id = services.server_id").
		Where("servers.project_id = ?", projectID).
		Where("service_log_sources.id = ?", id).
		First(&it).Error
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogSourceRepository) Create(ctx context.Context, it *model.ServiceLogSource) error {
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *LogSourceRepository) Save(ctx context.Context, it *model.ServiceLogSource) error {
	return r.db.WithContext(ctx).Save(it).Error
}

func (r *LogSourceRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ServiceLogSource{}, id).Error
}

func (r *LogSourceRepository) List(ctx context.Context, p LogSourceListParams) ([]model.ServiceLogSource, int64, error) {
	// join services + servers to apply project filter
	q := r.db.WithContext(ctx).
		Model(&model.ServiceLogSource{}).
		Joins("JOIN services ON services.id = service_log_sources.service_id").
		Joins("JOIN servers ON servers.id = services.server_id").
		Where("servers.project_id = ?", p.ProjectID)
	if p.ServiceID != nil && *p.ServiceID > 0 {
		q = q.Where("service_log_sources.service_id = ?", *p.ServiceID)
	}
	var list []model.ServiceLogSource
	total, err := listWithPagination(q, p.Page, p.PageSize, "service_log_sources.id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *LogSourceRepository) ListByProjectAndServer(ctx context.Context, projectID, serverID uint) ([]model.ServiceLogSource, error) {
	var list []model.ServiceLogSource
	err := r.db.WithContext(ctx).
		Model(&model.ServiceLogSource{}).
		Select("service_log_sources.*").
		Joins("JOIN services ON services.id = service_log_sources.service_id").
		Joins("JOIN servers ON servers.id = services.server_id").
		Where("servers.project_id = ?", projectID).
		Where("services.server_id = ?", serverID).
		Where("services.status = ?", model.StatusEnabled).
		Where("service_log_sources.status = ?", model.StatusEnabled).
		Order("service_log_sources.id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
