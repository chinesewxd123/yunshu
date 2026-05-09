package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type K8sNamespaceDenyRepository struct {
	db *gorm.DB
}

func NewK8sNamespaceDenyRepository(db *gorm.DB) *K8sNamespaceDenyRepository {
	return &K8sNamespaceDenyRepository{db: db}
}

// IsDenied 若用户任一角色码在该集群下配置了禁止该命名空间，则返回 true。
func (r *K8sNamespaceDenyRepository) IsDenied(ctx context.Context, roleCodes []string, clusterID uint, namespace string) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	ns := strings.TrimSpace(namespace)
	if len(roleCodes) == 0 || clusterID == 0 || ns == "" || ns == "_cluster" {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&model.K8sNamespaceDenyRule{}).
		Where("cluster_id = ? AND namespace = ? AND role_code IN ?", clusterID, ns, roleCodes).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// List 按可选条件列出规则。
func (r *K8sNamespaceDenyRepository) List(ctx context.Context, roleCode string, clusterID uint) ([]model.K8sNamespaceDenyRule, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceDenyRule{}).Order("cluster_id ASC, role_code ASC, namespace ASC")
	if rc := strings.TrimSpace(roleCode); rc != "" {
		q = q.Where("role_code = ?", rc)
	}
	if clusterID > 0 {
		q = q.Where("cluster_id = ?", clusterID)
	}
	var list []model.K8sNamespaceDenyRule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Create 插入一条规则（唯一约束冲突由调用方处理）。
func (r *K8sNamespaceDenyRepository) Create(ctx context.Context, it *model.K8sNamespaceDenyRule) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(it).Error
}

// DeleteByID 按主键删除。
func (r *K8sNamespaceDenyRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Delete(&model.K8sNamespaceDenyRule{}, id).Error
}
