package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type ServerGroupRepository struct {
	db *gorm.DB
}

func NewServerGroupRepository(db *gorm.DB) *ServerGroupRepository { return &ServerGroupRepository{db: db} }

func (r *ServerGroupRepository) Create(ctx context.Context, item *model.ServerGroup) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *ServerGroupRepository) Save(ctx context.Context, item *model.ServerGroup) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *ServerGroupRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ServerGroup{}, id).Error
}

func (r *ServerGroupRepository) GetByID(ctx context.Context, id uint) (*model.ServerGroup, error) {
	var item model.ServerGroup
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ServerGroupRepository) ListByProject(ctx context.Context, projectID uint) ([]model.ServerGroup, error) {
	var list []model.ServerGroup
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sort ASC, id ASC").
		Find(&list).Error
	return list, err
}
