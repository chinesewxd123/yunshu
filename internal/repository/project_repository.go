package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

type ProjectListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository { return &ProjectRepository{db: db} }

func (r *ProjectRepository) Create(ctx context.Context, p *model.Project) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *ProjectRepository) Save(ctx context.Context, p *model.Project) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *ProjectRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Project{}, id).Error
}

func (r *ProjectRepository) GetByID(ctx context.Context, id uint) (*model.Project, error) {
	var p model.Project
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) List(ctx context.Context, params ProjectListParams) ([]model.Project, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Project{})
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		q = q.Where("name LIKE ? OR code LIKE ?", kw, kw)
	}
	var list []model.Project
	total, err := listWithPagination(q, params.Page, params.PageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListVisibleToUser 仅返回指定用户作为成员参与的项目（与 project_members 联表）。
func (r *ProjectRepository) ListVisibleToUser(ctx context.Context, userID uint, params ProjectListParams) ([]model.Project, int64, error) {
	if userID == 0 {
		return nil, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id IN (?)", r.db.Model(&model.ProjectMember{}).Select("project_id").Where("user_id = ?", userID))
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		q = q.Where("name LIKE ? OR code LIKE ?", kw, kw)
	}
	var list []model.Project
	total, err := listWithPagination(q, params.Page, params.PageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
