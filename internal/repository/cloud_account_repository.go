package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type CloudAccountRepository struct {
	db *gorm.DB
}

func NewCloudAccountRepository(db *gorm.DB) *CloudAccountRepository {
	return &CloudAccountRepository{db: db}
}

func (r *CloudAccountRepository) Create(ctx context.Context, item *model.CloudAccount) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *CloudAccountRepository) Save(ctx context.Context, item *model.CloudAccount) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *CloudAccountRepository) GetByID(ctx context.Context, id uint) (*model.CloudAccount, error) {
	var item model.CloudAccount
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CloudAccountRepository) ListByProjectAndGroup(ctx context.Context, projectID uint, groupID *uint) ([]model.CloudAccount, error) {
	q := r.db.WithContext(ctx).Model(&model.CloudAccount{}).Where("project_id = ?", projectID)
	if groupID != nil {
		q = q.Where("group_id = ?", *groupID)
	}
	var list []model.CloudAccount
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

func (r *CloudAccountRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.CloudAccount{}, id).Error
}
