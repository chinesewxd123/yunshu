package repository

import (
	"context"
	"time"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type AgentDiscoveryRepository struct {
	db *gorm.DB
}

func NewAgentDiscoveryRepository(db *gorm.DB) *AgentDiscoveryRepository { return &AgentDiscoveryRepository{db: db} }

func (r *AgentDiscoveryRepository) UpsertMany(ctx context.Context, projectID, serverID uint, items []model.AgentDiscovery) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now()
	for i := range items {
		items[i].ProjectID = projectID
		items[i].ServerID = serverID
		if items[i].FirstSeenAt.IsZero() {
			items[i].FirstSeenAt = now
		}
		items[i].LastSeenAt = now
	}

	// MySQL-compatible "upsert": query then update/insert per row.
	// Keeping it simple to avoid DB-specific ON DUPLICATE KEY dependencies.
	for _, it := range items {
		var existing model.AgentDiscovery
		err := r.db.WithContext(ctx).
			Where("project_id=? AND server_id=? AND kind=? AND value=?", projectID, serverID, it.Kind, it.Value).
			First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			if err := r.db.WithContext(ctx).Create(&it).Error; err != nil {
				return err
			}
			continue
		}
		updates := map[string]any{
			"last_seen_at": it.LastSeenAt,
			"extra":        it.Extra,
		}
		if err := r.db.WithContext(ctx).Model(&model.AgentDiscovery{}).Where("id=?", existing.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *AgentDiscoveryRepository) List(ctx context.Context, projectID, serverID uint, kind *string, limit int) ([]model.AgentDiscovery, error) {
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	q := r.db.WithContext(ctx).Model(&model.AgentDiscovery{}).
		Where("project_id=? AND server_id=?", projectID, serverID).
		Order("last_seen_at DESC").
		Limit(limit)
	if kind != nil && *kind != "" {
		q = q.Where("kind=?", *kind)
	}
	var out []model.AgentDiscovery
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

