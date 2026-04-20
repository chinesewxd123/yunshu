package repository

import (
	"context"
	"fmt"
	"strings"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

type UserListParams struct {
	Keyword       string
	DepartmentID  *uint
	DepartmentIDs []uint
	OnlyUserID    *uint
	Page          int
	PageSize      int
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
	err := r.db.WithContext(ctx).Preload("Roles").Preload("Department").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").Preload("Department").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").Preload("Department").Where("email = ?", email).First(&user).Error
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
	if params.OnlyUserID != nil && *params.OnlyUserID > 0 {
		query = query.Where("id = ?", *params.OnlyUserID)
	}
	if params.DepartmentID != nil && *params.DepartmentID > 0 {
		query = query.Where("department_id = ?", *params.DepartmentID)
	}
	if len(params.DepartmentIDs) > 0 {
		query = query.Where("department_id IN ?", params.DepartmentIDs)
	}

	var users []model.User
	query = query.Preload("Roles").Preload("Department")
	total, err := listWithPagination(query, params.Page, params.PageSize, "id DESC", &users)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepository) ReplaceRoles(ctx context.Context, user *model.User, roles []model.Role) error {
	return r.db.WithContext(ctx).Model(user).Association("Roles").Replace(roles)
}

func (r *UserRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ? OR email = ?", username, email).Count(&count)
	return count > 0, err.Error
}

// ListAll returns all users without pagination. Used for export.
func (r *UserRepository) ListAll(ctx context.Context) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).Preload("Roles").Preload("Department").Order("id DESC").Find(&users).Error
	return users, err
}

func (r *UserRepository) ListByIDs(ctx context.Context, ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	var users []model.User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// ListActiveIDsByDepartmentIDs 返回部门 ID 列表下、状态为启用的用户 ID（精确匹配 department_id）。
func (r *UserRepository) ListActiveIDsByDepartmentIDs(ctx context.Context, deptIDs []uint) ([]uint, error) {
	if len(deptIDs) == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("department_id IN ? AND status = ?", deptIDs, 1).
		Distinct().
		Pluck("id", &ids).Error
	return ids, err
}

// ListActiveIDsByDepartmentSubtree 将 rootDeptIDs 视为根部门，包含其所有子部门（物化路径 ancestors 含 /rootId/）下的启用用户。
func (r *UserRepository) ListActiveIDsByDepartmentSubtree(ctx context.Context, rootDeptIDs []uint) ([]uint, error) {
	if len(rootDeptIDs) == 0 {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.User{}).
		Distinct("users.id").
		Joins("INNER JOIN departments ON users.department_id = departments.id AND departments.deleted_at IS NULL").
		Where("users.status = ? AND users.deleted_at IS NULL", 1)

	var orParts []string
	var args []interface{}
	orParts = append(orParts, "departments.id IN ?")
	args = append(args, rootDeptIDs)
	for _, rid := range rootDeptIDs {
		if rid == 0 {
			continue
		}
		orParts = append(orParts, "departments.ancestors LIKE ?")
		args = append(args, fmt.Sprintf("%%/%d/%%", rid))
	}
	q = q.Where("("+strings.Join(orParts, " OR ")+")", args...)

	var ids []uint
	err := q.Pluck("users.id", &ids).Error
	return ids, err
}
