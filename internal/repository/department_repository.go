package repository

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) DB() *gorm.DB {
	return r.db
}

func (r *DepartmentRepository) Create(ctx context.Context, item *model.Department) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *DepartmentRepository) Save(ctx context.Context, item *model.Department) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *DepartmentRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Department{}, id).Error
}

func (r *DepartmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	var item model.Department
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DepartmentRepository) GetByCode(ctx context.Context, code string) (*model.Department, error) {
	var item model.Department
	if err := r.db.WithContext(ctx).Where("code = ?", strings.TrimSpace(code)).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DepartmentRepository) ListAll(ctx context.Context) ([]model.Department, error) {
	var list []model.Department
	err := r.db.WithContext(ctx).
		Order("level ASC, sort ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *DepartmentRepository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.Department{}).Where("code = ?", strings.TrimSpace(code))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DepartmentRepository) ExistsByNameInParent(ctx context.Context, parentID *uint, name string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.Department{}).Where("name = ?", strings.TrimSpace(name))
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DepartmentRepository) CountChildren(ctx context.Context, id uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Department{}).Where("parent_id = ?", id).Count(&count).Error
	return count, err
}

func (r *DepartmentRepository) CountUsers(ctx context.Context, id uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("department_id = ?", id).Count(&count).Error
	return count, err
}

func (r *DepartmentRepository) ListDescendantIDsAndSelf(ctx context.Context, id uint) ([]uint, error) {
	item, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	prefix := item.Ancestors + strings.TrimSpace(fmt.Sprintf("%d", item.ID)) + "/"
	ids := make([]uint, 0)
	if err := r.db.WithContext(ctx).
		Model(&model.Department{}).
		Where("id = ? OR ancestors LIKE ?", id, prefix+"%").
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
