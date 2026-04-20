package repository

import (
	"context"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/pagination"

	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

type PermissionListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *PermissionRepository) Save(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

func (r *PermissionRepository) Delete(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Delete(permission).Error
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uint) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepository) List(ctx context.Context, params PermissionListParams) ([]model.Permission, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Permission{})
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR resource LIKE ? OR action LIKE ?", keyword, keyword, keyword)
	}
	var permissions []model.Permission
	page, pageSize := pagination.Normalize(params.Page, params.PageSize)
	total, err := listWithPagination(query, page, pageSize, "id DESC", &permissions)
	if err != nil {
		return nil, 0, err
	}
	return permissions, total, nil
}

func (r *PermissionRepository) ListAll(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
