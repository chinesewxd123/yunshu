package repository

import (
	"context"
	"strings"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type ServerRepository struct {
	db *gorm.DB
}

type ServerListParams struct {
	ProjectID  uint
	Keyword    string
	GroupID    *uint
	SourceType string
	Provider   string
	Page       int
	PageSize   int
}

func NewServerRepository(db *gorm.DB) *ServerRepository { return &ServerRepository{db: db} }

func (r *ServerRepository) Create(ctx context.Context, s *model.Server) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *ServerRepository) Save(ctx context.Context, s *model.Server) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *ServerRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Server{}, id).Error
}

func (r *ServerRepository) GetByID(ctx context.Context, id uint) (*model.Server, error) {
	var s model.Server
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ProjectNameByID returns the project display name for list views (e.g. agent list per project).
func (r *ServerRepository) ProjectNameByID(ctx context.Context, projectID uint) (string, error) {
	var p model.Project
	if err := r.db.WithContext(ctx).Select("name").First(&p, projectID).Error; err != nil {
		return "", err
	}
	return strings.TrimSpace(p.Name), nil
}

func (r *ServerRepository) List(ctx context.Context, params ServerListParams) ([]model.Server, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Server{}).Where("project_id = ?", params.ProjectID)
	if params.GroupID != nil {
		q = q.Where("group_id = ?", *params.GroupID)
	}
	if params.SourceType != "" {
		if params.SourceType == model.ServerGroupCategorySelfHosted {
			// Backward compatibility for historical rows that have empty source_type.
			q = q.Where("(source_type = ? OR source_type = '' OR source_type IS NULL)", params.SourceType)
		} else {
			q = q.Where("source_type = ?", params.SourceType)
		}
	}
	if params.Provider != "" {
		q = q.Where("provider = ?", params.Provider)
	}
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		q = q.Where("name LIKE ? OR host LIKE ? OR tags LIKE ?", kw, kw, kw)
	}
	var list []model.Server
	total, err := listWithPagination(q, params.Page, params.PageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ServerRepository) GetByProjectProviderInstance(ctx context.Context, projectID uint, provider, cloudInstanceID string) (*model.Server, error) {
	var s model.Server
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND provider = ? AND cloud_instance_id = ?", projectID, provider, cloudInstanceID).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServerRepository) ListByProjectWithoutGroup(ctx context.Context, projectID uint) ([]model.Server, error) {
	var list []model.Server
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND group_id IS NULL", projectID).
		Find(&list).Error
	return list, err
}

func (r *ServerRepository) ListByProjectGroupProvider(ctx context.Context, projectID, groupID uint, provider string) ([]model.Server, error) {
	var list []model.Server
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND group_id = ? AND source_type = ? AND provider = ?", projectID, groupID, model.ServerGroupCategoryCloud, provider).
		Find(&list).Error
	return list, err
}

func (r *ServerRepository) ListByProjectProviderCloud(ctx context.Context, projectID uint, provider string) ([]model.Server, error) {
	var list []model.Server
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND source_type = ? AND provider = ?", projectID, model.ServerGroupCategoryCloud, provider).
		Find(&list).Error
	return list, err
}

func (r *ServerRepository) UpsertCredential(ctx context.Context, cred *model.ServerCredential) error {
	// server_id is uniqueIndex; Save will update if primary key exists but might insert new.
	// We do: find existing by server_id then save.
	var existing model.ServerCredential
	err := r.db.WithContext(ctx).Where("server_id = ?", cred.ServerID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(cred).Error
	}
	cred.ID = existing.ID
	cred.CreatedAt = existing.CreatedAt
	cred.DeletedAt = existing.DeletedAt
	return r.db.WithContext(ctx).Save(cred).Error
}

func (r *ServerRepository) GetCredentialByServerID(ctx context.Context, serverID uint) (*model.ServerCredential, error) {
	var c model.ServerCredential
	if err := r.db.WithContext(ctx).Where("server_id = ?", serverID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}
