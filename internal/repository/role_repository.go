package repository

import (
	"context"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

type RoleListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *RoleRepository) Save(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *RoleRepository) Delete(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Delete(role).Error
}

func (r *RoleRepository) GetByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.Role, error) {
	if len(ids) == 0 {
		return []model.Role{}, nil
	}

	var roles []model.Role
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) List(ctx context.Context, params RoleListParams) ([]model.Role, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Role{})
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var roles []model.Role
	err := query.Order("id DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Find(&roles).Error
	if err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (r *RoleRepository) ListAll(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
