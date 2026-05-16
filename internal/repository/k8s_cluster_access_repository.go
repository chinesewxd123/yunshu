package repository

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type K8sClusterAccessRepository struct {
	db *gorm.DB
}

func NewK8sClusterAccessRepository(db *gorm.DB) *K8sClusterAccessRepository {
	return &K8sClusterAccessRepository{db: db}
}

// Upsert 按 (principal_kind, principal_ref, cluster_id) 幂等写入或更新档位。
func (r *K8sClusterAccessRepository) Upsert(ctx context.Context, it *model.K8sClusterAccessGrant) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "principal_kind"}, {Name: "principal_ref"}, {Name: "cluster_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"preset", "updated_at"}),
	}).Create(it).Error
}

// ListByPrincipal 列出某主体全部集群档位。
// ListGrantsApplyingToCluster 列出对该集群生效的档位（含 cluster_id=0 的全局授权）。
func (r *K8sClusterAccessRepository) ListGrantsApplyingToCluster(ctx context.Context, clusterID uint) ([]model.K8sClusterAccessGrant, error) {
	if r == nil || r.db == nil || clusterID == 0 {
		return []model.K8sClusterAccessGrant{}, nil
	}
	var list []model.K8sClusterAccessGrant
	err := r.db.WithContext(ctx).Model(&model.K8sClusterAccessGrant{}).
		Where("cluster_id = ? OR cluster_id = 0", clusterID).
		Order("principal_kind ASC, principal_ref ASC, cluster_id ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *K8sClusterAccessRepository) ListByPrincipal(ctx context.Context, kind, ref string) ([]model.K8sClusterAccessGrant, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	k := strings.TrimSpace(kind)
	rc := strings.TrimSpace(ref)
	if k == "" || rc == "" {
		return []model.K8sClusterAccessGrant{}, nil
	}
	var list []model.K8sClusterAccessGrant
	err := r.db.WithContext(ctx).Model(&model.K8sClusterAccessGrant{}).
		Where("principal_kind = ? AND principal_ref = ?", k, rc).
		Order("cluster_id ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByRoleCode 兼容：按角色码列出。
func (r *K8sClusterAccessRepository) ListByRoleCode(ctx context.Context, roleCode string) ([]model.K8sClusterAccessGrant, error) {
	return r.ListByPrincipal(ctx, model.K8sPrincipalRole, strings.TrimSpace(roleCode))
}

func (r *K8sClusterAccessRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Delete(&model.K8sClusterAccessGrant{}, id).Error
}

// EffectiveTier 返回主体集合在指定集群上的最高档位。
func (r *K8sClusterAccessRepository) EffectiveTier(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) int {
	if r == nil || r.db == nil {
		return 0
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return 0
	}
	q := r.db.WithContext(ctx).Model(&model.K8sClusterAccessGrant{})
	var parts []string
	var args []any
	for _, row := range rows {
		parts = append(parts, "(principal_kind = ? AND principal_ref = ?)")
		args = append(args, row.Kind, row.Ref)
	}
	q = q.Where(strings.Join(parts, " OR "), args...)
	if clusterID > 0 {
		q = q.Where("cluster_id = 0 OR cluster_id = ?", clusterID)
	} else {
		q = q.Where("cluster_id = 0")
	}
	var presets []string
	if err := q.Pluck("preset", &presets).Error; err != nil || len(presets) == 0 {
		return 0
	}
	maxV := 0
	for _, p := range presets {
		if v := k8sPresetRank(p); v > maxV {
			maxV = v
		}
	}
	return maxV
}

// EffectiveTierIndex 主体在各集群上的档位（cluster_id=0 表示全局档位）。
type EffectiveTierIndex struct {
	GlobalRank int
	PerCluster map[uint]int
}

// BuildEffectiveTierIndex 一次加载主体相关的全部档位授权，供批量过滤使用。
func (r *K8sClusterAccessRepository) BuildEffectiveTierIndex(ctx context.Context, pack k8sauth.PrincipalPack) (EffectiveTierIndex, error) {
	idx := EffectiveTierIndex{PerCluster: map[uint]int{}}
	if r == nil || r.db == nil {
		return idx, nil
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return idx, nil
	}
	q := r.db.WithContext(ctx).Model(&model.K8sClusterAccessGrant{})
	var parts []string
	var args []any
	for _, row := range rows {
		parts = append(parts, "(principal_kind = ? AND principal_ref = ?)")
		args = append(args, row.Kind, row.Ref)
	}
	var grants []model.K8sClusterAccessGrant
	if err := q.Where(strings.Join(parts, " OR "), args...).Find(&grants).Error; err != nil {
		return idx, err
	}
	for _, g := range grants {
		rank := k8sPresetRank(g.Preset)
		if g.ClusterID == 0 {
			if rank > idx.GlobalRank {
				idx.GlobalRank = rank
			}
			continue
		}
		if rank > idx.PerCluster[g.ClusterID] {
			idx.PerCluster[g.ClusterID] = rank
		}
	}
	return idx, nil
}

// ClusterAccessible 判断主体在指定集群上是否达到 minRank（含全局 cluster_id=0 授权）。
func (idx EffectiveTierIndex) ClusterAccessible(clusterID uint, minRank int) bool {
	if minRank <= 0 {
		return true
	}
	if idx.GlobalRank >= minRank {
		return true
	}
	return idx.PerCluster[clusterID] >= minRank
}

// HasAnyK8sGrant 是否存在任意集群档位。
func (r *K8sClusterAccessRepository) HasAnyK8sGrant(ctx context.Context, pack k8sauth.PrincipalPack) bool {
	if r == nil || r.db == nil {
		return false
	}
	rows := pack.PrincipalRows()
	if len(rows) == 0 {
		return false
	}
	q := r.db.WithContext(ctx).Model(&model.K8sClusterAccessGrant{})
	var parts []string
	var args []any
	for _, row := range rows {
		parts = append(parts, "(principal_kind = ? AND principal_ref = ?)")
		args = append(args, row.Kind, row.Ref)
	}
	q = q.Where(strings.Join(parts, " OR "), args...)
	var n int64
	if err := q.Limit(1).Count(&n).Error; err != nil {
		return false
	}
	return n > 0
}

func k8sPresetRank(p string) int {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "readonly":
		return 1
	case "readonly_exec":
		return 2
	case "admin":
		return 3
	default:
		return 0
	}
}

// ResolvePrincipalRef 解析 principal_ref。
func ResolvePrincipalRef(kind string, id uint, code string) (string, error) {
	k := strings.TrimSpace(strings.ToLower(kind))
	switch k {
	case model.K8sPrincipalRole:
		s := strings.TrimSpace(code)
		if s == "" {
			return "", fmt.Errorf("role code required")
		}
		return s, nil
	case model.K8sPrincipalUser:
		if id == 0 {
			return "", fmt.Errorf("user id required")
		}
		return k8sauth.UserRefString(id), nil
	case model.K8sPrincipalGroup:
		s := strings.TrimSpace(code)
		if s == "" {
			return "", fmt.Errorf("group code required")
		}
		return s, nil
	default:
		return "", fmt.Errorf("invalid principal_kind")
	}
}
