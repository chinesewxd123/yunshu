package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type UserGroupRepository struct {
	db *gorm.DB
}

type UserGroupListParams struct {
	Keyword         string
	Page            int
	PageSize        int
	ScopeProjectID  *uint // 非空时仅返回全局组或该项目的组
}

func NewUserGroupRepository(db *gorm.DB) *UserGroupRepository {
	return &UserGroupRepository{db: db}
}

func (r *UserGroupRepository) GetByID(ctx context.Context, id uint) (*model.UserGroup, error) {
	if r == nil || r.db == nil || id == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var g model.UserGroup
	if err := r.db.WithContext(ctx).First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *UserGroupRepository) GetByCode(ctx context.Context, code string) (*model.UserGroup, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	c := strings.TrimSpace(code)
	if c == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var g model.UserGroup
	if err := r.db.WithContext(ctx).Where("code = ?", c).First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *UserGroupRepository) Create(ctx context.Context, g *model.UserGroup) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *UserGroupRepository) Save(ctx context.Context, g *model.UserGroup) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *UserGroupRepository) Delete(ctx context.Context, g *model.UserGroup) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Delete(g).Error
}

func (r *UserGroupRepository) List(ctx context.Context, params UserGroupListParams) ([]model.UserGroup, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&model.UserGroup{})
	if kw := strings.TrimSpace(params.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if params.ScopeProjectID != nil && *params.ScopeProjectID > 0 {
		sp := *params.ScopeProjectID
		q = q.Where("(scope_project_id IS NULL OR scope_project_id = ?)", sp)
	}
	var list []model.UserGroup
	total, err := listWithPagination(q, params.Page, params.PageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *UserGroupRepository) ListMemberUserIDs(ctx context.Context, groupID uint) ([]uint, error) {
	if r == nil || r.db == nil || groupID == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.UserGroupUser{}).
		Where("user_group_id = ?", groupID).
		Order("user_id ASC").
		Pluck("user_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *UserGroupRepository) CountMembers(ctx context.Context, groupID uint) (int64, error) {
	if r == nil || r.db == nil || groupID == 0 {
		return 0, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&model.UserGroupUser{}).Where("user_group_id = ?", groupID).Count(&n).Error
	return n, err
}

// ReplaceMemberUserIDs 全量替换组成员（空列表表示清空）。
func (r *UserGroupRepository) ReplaceMemberUserIDs(ctx context.Context, groupID uint, userIDs []uint) error {
	if r == nil || r.db == nil || groupID == 0 {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_group_id = ?", groupID).Delete(&model.UserGroupUser{}).Error; err != nil {
			return err
		}
		seen := map[uint]struct{}{}
		for _, uid := range userIDs {
			if uid == 0 {
				continue
			}
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
			row := model.UserGroupUser{UserID: uid, UserGroupID: groupID}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
