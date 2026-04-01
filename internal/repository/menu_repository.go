package repository

import (
	"context"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type MenuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) *MenuRepository {
	return &MenuRepository{db: db}
}

func (r *MenuRepository) Create(ctx context.Context, menu *model.Menu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *MenuRepository) GetByID(ctx context.Context, id uint) (*model.Menu, error) {
	var menu model.Menu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepository) Update(ctx context.Context, menu *model.Menu) error {
	return r.db.WithContext(ctx).Save(menu).Error
}

func (r *MenuRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Menu{}, id).Error
}

func (r *MenuRepository) ListAll(ctx context.Context) ([]model.Menu, error) {
	var list []model.Menu
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *MenuRepository) Tree(ctx context.Context) ([]model.Menu, error) {
	list, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(list, 0), nil
}

func buildMenuTree(menus []model.Menu, parentID uint) []model.Menu {
	var tree []model.Menu
	for _, m := range menus {
		if m.ParentID != nil && *m.ParentID == parentID || m.ParentID == nil && parentID == 0 && hasChildren(menus, m.ID) {
			m.Children = buildMenuTree(menus, m.ID)
			tree = append(tree, m)
		} else if m.ParentID == nil && parentID == 0 && !hasChildren(menus, m.ID) {
			tree = append(tree, m)
		}
	}
	return tree
}

func hasChildren(menus []model.Menu, id uint) bool {
	for _, m := range menus {
		if m.ParentID != nil && *m.ParentID == id {
			return true
		}
	}
	return false
}

func (r *MenuRepository) CountChildren(ctx context.Context, parentID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Menu{}).Where("parent_id = ?", parentID).Count(&count)
	return count, err.Error
}
