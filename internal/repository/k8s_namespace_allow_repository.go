package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"

	"gorm.io/gorm"
)

type K8sNamespaceAllowRepository struct {
	db *gorm.DB
}

func NewK8sNamespaceAllowRepository(db *gorm.DB) *K8sNamespaceAllowRepository {
	return &K8sNamespaceAllowRepository{db: db}
}

// WhitelistActiveForCluster 若任一主体在该集群配置了白名单规则，则进入「仅允许名单内 NS」模式。
func (r *K8sNamespaceAllowRepository) WhitelistActiveForCluster(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) (bool, error) {
	if r == nil || r.db == nil || clusterID == 0 {
		return false, nil
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return false, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceAllowRule{}).
		Where("(cluster_id = ? OR cluster_id = 0)", clusterID)
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

// NamespaceAllowed 在白名单模式下，命名空间是否在允许列表中（任一主体允许即可）。
func (r *K8sNamespaceAllowRepository) NamespaceAllowed(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	ns := strings.TrimSpace(namespace)
	if clusterID == 0 || ns == "" || ns == "_cluster" {
		return false, nil
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return false, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceAllowRule{}).
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

func (r *K8sNamespaceAllowRepository) List(ctx context.Context, principalKind, principalRef string, clusterID uint) ([]model.K8sNamespaceAllowRule, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sNamespaceAllowRule{}).Order("cluster_id ASC, principal_kind ASC, principal_ref ASC, namespace ASC")
	if k := strings.TrimSpace(principalKind); k != "" {
		q = q.Where("principal_kind = ?", k)
	}
	if ref := strings.TrimSpace(principalRef); ref != "" {
		q = q.Where("principal_ref = ?", ref)
	}
	if clusterID > 0 {
		q = q.Where("cluster_id = ?", clusterID)
	}
	var list []model.K8sNamespaceAllowRule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *K8sNamespaceAllowRepository) Create(ctx context.Context, it *model.K8sNamespaceAllowRule) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *K8sNamespaceAllowRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Delete(&model.K8sNamespaceAllowRule{}, id).Error
}
