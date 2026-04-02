package repository

import (
	"context"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/pagination"

	"gorm.io/gorm"
)

type OperationLogListParams struct {
	Method     string
	Path       string
	StatusCode *int
	Page       int
	PageSize   int
}

type OperationLogRepository struct {
	db *gorm.DB
}

func NewOperationLogRepository(db *gorm.DB) *OperationLogRepository {
	return &OperationLogRepository{db: db}
}

func (r *OperationLogRepository) Create(ctx context.Context, log *model.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *OperationLogRepository) List(ctx context.Context, p OperationLogListParams) ([]model.OperationLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.OperationLog{})
	if p.Method != "" {
		q = q.Where("method = ?", p.Method)
	}
	if p.Path != "" {
		q = q.Where("path LIKE ?", "%"+p.Path+"%")
	}
	if p.StatusCode != nil {
		q = q.Where("status_code = ?", *p.StatusCode)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	offset := (page - 1) * pageSize

	var list []model.OperationLog
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OperationLogRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.OperationLog{}, id).Error
}

func (r *OperationLogRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.OperationLog{}).Error
}
