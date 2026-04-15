package repository

import (
	"context"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type LoginLogListParams struct {
	Username string
	Status   *int
	Source   string
	Page     int
	PageSize int
}

type LoginLogRepository struct {
	db *gorm.DB
}

func NewLoginLogRepository(db *gorm.DB) *LoginLogRepository {
	return &LoginLogRepository{db: db}
}

func (r *LoginLogRepository) Create(ctx context.Context, log *model.LoginLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *LoginLogRepository) List(ctx context.Context, p LoginLogListParams) ([]model.LoginLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LoginLog{})
	if p.Username != "" {
		q = q.Where("username LIKE ?", "%"+p.Username+"%")
	}
	if p.Status != nil {
		q = q.Where("status = ?", *p.Status)
	}
	if p.Source != "" {
		q = q.Where("source = ?", p.Source)
	}

	var list []model.LoginLog
	total, err := listWithPagination(q, p.Page, p.PageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *LoginLogRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.LoginLog{}, id).Error
}

func (r *LoginLogRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.LoginLog{}).Error
}
