package repository

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type K8sClusterRepository struct {
	db *gorm.DB
}

type K8sClusterListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewK8sClusterRepository(db *gorm.DB) *K8sClusterRepository {
	return &K8sClusterRepository{db: db}
}

func (r *K8sClusterRepository) Create(ctx context.Context, cluster *model.K8sCluster) error {
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *K8sClusterRepository) Update(ctx context.Context, cluster *model.K8sCluster) error {
	return r.db.WithContext(ctx).Save(cluster).Error
}

func (r *K8sClusterRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.K8sCluster{}, id).Error
}

func (r *K8sClusterRepository) GetByID(ctx context.Context, id uint) (*model.K8sCluster, error) {
	var cluster model.K8sCluster
	err := r.db.WithContext(ctx).First(&cluster, id).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *K8sClusterRepository) List(ctx context.Context, params K8sClusterListParams) ([]model.K8sCluster, int64, error) {
	page, pageSize := pagination.Normalize(params.Page, params.PageSize)

	query := r.db.WithContext(ctx).Model(&model.K8sCluster{})
	if params.Keyword != "" {
		kw := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ?", kw)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.K8sCluster
	if err := query.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
