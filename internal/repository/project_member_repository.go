package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// ProjectMemberListRow 项目成员列表展示（联表 users）。
type ProjectMemberListRow struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Role      string    `json:"role"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectMemberRepository struct {
	db *gorm.DB
}

func NewProjectMemberRepository(db *gorm.DB) *ProjectMemberRepository {
	return &ProjectMemberRepository{db: db}
}

func (r *ProjectMemberRepository) ListByProject(ctx context.Context, projectID uint) ([]model.ProjectMember, error) {
	var list []model.ProjectMember
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListUserIDsByProject 返回项目下仍有效的成员用户 ID（去重顺序按 id）。
func (r *ProjectMemberRepository) ListUserIDsByProject(ctx context.Context, projectID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ?", projectID).
		Order("id ASC").
		Pluck("user_id", &ids).Error
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]struct{})
	var out []uint
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (r *ProjectMemberRepository) GetByID(ctx context.Context, id uint) (*model.ProjectMember, error) {
	var row model.ProjectMember
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ProjectMemberRepository) GetByProjectAndUser(ctx context.Context, projectID, userID uint) (*model.ProjectMember, error) {
	var row model.ProjectMember
	err := r.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, userID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ProjectMemberRepository) Create(ctx context.Context, row *model.ProjectMember) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ProjectMemberRepository) Save(ctx context.Context, row *model.ProjectMember) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *ProjectMemberRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ProjectMember{}, id).Error
}

func (r *ProjectMemberRepository) DeleteByProject(ctx context.Context, projectID uint) error {
	return r.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&model.ProjectMember{}).Error
}

// ListDisplayByProject 联表查询成员及用户基本信息（用于项目成员管理页）。
func (r *ProjectMemberRepository) ListDisplayByProject(ctx context.Context, projectID uint) ([]ProjectMemberListRow, error) {
	var rows []ProjectMemberListRow
	err := r.db.WithContext(ctx).Table(model.ProjectMember{}.TableName()+" AS pm").
		Select("pm.id, pm.user_id, pm.role, pm.created_at, u.username, u.nickname, u.email").
		Joins("INNER JOIN users u ON u.id = pm.user_id AND u.deleted_at IS NULL").
		Where("pm.project_id = ? AND pm.deleted_at IS NULL", projectID).
		Order("pm.id ASC").
		Scan(&rows).Error
	return rows, err
}
