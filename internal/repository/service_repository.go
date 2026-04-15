package repository

import (
	"context"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type ServiceRepository struct{ db *gorm.DB }

func NewServiceRepository(db *gorm.DB) *ServiceRepository { return &ServiceRepository{db: db} }

type ServiceListParams struct {
	ProjectID uint
	ServerID  *uint
	Keyword   string
	Page      int
	PageSize  int
}

func (r *ServiceRepository) GetByID(ctx context.Context, id uint) (*model.Service, error) {
	var s model.Service
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServiceRepository) Create(ctx context.Context, s *model.Service) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *ServiceRepository) Save(ctx context.Context, s *model.Service) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *ServiceRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Service{}, id).Error
}

func (r *ServiceRepository) List(ctx context.Context, p ServiceListParams) ([]model.Service, int64, error) {
	// join servers to apply project filter
	q := r.db.WithContext(ctx).
		Model(&model.Service{}).
		Joins("JOIN servers ON servers.id = services.server_id").
		Where("servers.project_id = ?", p.ProjectID)
	if p.ServerID != nil && *p.ServerID > 0 {
		q = q.Where("services.server_id = ?", *p.ServerID)
	}
	if p.Keyword != "" {
		kw := "%" + p.Keyword + "%"
		q = q.Where("services.name LIKE ? OR services.labels LIKE ?", kw, kw)
	}
	var list []model.Service
	total, err := listWithPagination(q, p.Page, p.PageSize, "services.id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
