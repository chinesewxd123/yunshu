package repository

import (
	"context"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

type UserListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) Save(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) Delete(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Delete(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) List(ctx context.Context, params UserListParams) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{})
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", keyword, keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := query.Preload("Roles").
		Order("id DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepository) ReplaceRoles(ctx context.Context, user *model.User, roles []model.Role) error {
	return r.db.WithContext(ctx).Model(user).Association("Roles").Replace(roles)
}
