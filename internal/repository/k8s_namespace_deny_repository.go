package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"

	"gorm.io/gorm"
)

type K8sNamespaceDenyRepository struct {
	db *gorm.DB
}

func NewK8sNamespaceDenyRepository(db *gorm.DB) *K8sNamespaceDenyRepository {
	return &K8sNamespaceDenyRepository{db: db}
}

// IsDenied 任一主体在该集群下禁止该命名空间则 true。
// cluster_id=0 的规则视为「全部集群」通配，与 k8s_cluster_access_grants 语义一致。
func (r *K8sNamespaceDenyRepository) IsDenied(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	ns := strings.TrimSpace(namespace)
	rows := pack.PrincipalRows()
	if len(rows) == 0 || ns == "" || ns == "_cluster" {
		return false, nil
	}
	if clusterID == 0 {
		return false, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceDenyRule{}).
		Where("(cluster_id = ? OR cluster_id = 0) AND namespace = ?", clusterID, ns)
	var parts []string
	var args []any
	for _, row := range rows {
		parts = append(parts, "(principal_kind = ? AND principal_ref = ?)")
		args = append(args, row.Kind, row.Ref)
	}
	q = q.Where(strings.Join(parts, " OR "), args...)
	var n int64
	if err := q.Limit(1).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeniedNamespaceNames 返回当前主体集合在该集群（含 cluster_id=0 通配规则）下被禁止访问的命名空间名（去重、有序）。
func (r *K8sNamespaceDenyRepository) DeniedNamespaceNames(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) ([]string, error) {
	if r == nil || r.db == nil || clusterID == 0 {
		return nil, nil
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceDenyRule{}).
		Where("cluster_id = ? OR cluster_id = ?", clusterID, 0)
	var parts []string
	var args []any
	for _, row := range rows {
		parts = append(parts, "(principal_kind = ? AND principal_ref = ?)")
		args = append(args, row.Kind, row.Ref)
	}
	q = q.Where(strings.Join(parts, " OR "), args...)
	var names []string
	if err := q.Distinct("namespace").Order("namespace ASC").Pluck("namespace", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}

// List 按可选条件列出规则。
func (r *K8sNamespaceDenyRepository) List(ctx context.Context, principalKind, principalRef string, clusterID uint) ([]model.K8sNamespaceDenyRule, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceDenyRule{}).Order("cluster_id ASC, principal_kind ASC, principal_ref ASC, namespace ASC")
	if k := strings.TrimSpace(principalKind); k != "" {
		q = q.Where("principal_kind = ?", k)
	}
	if ref := strings.TrimSpace(principalRef); ref != "" {
		q = q.Where("principal_ref = ?", ref)
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

func (r *K8sNamespaceDenyRepository) Create(ctx context.Context, it *model.K8sNamespaceDenyRule) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *K8sNamespaceDenyRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Delete(&model.K8sNamespaceDenyRule{}, id).Error
}
