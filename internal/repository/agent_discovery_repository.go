package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentDiscoveryRepository struct {
	db *gorm.DB
}

func NewAgentDiscoveryRepository(db *gorm.DB) *AgentDiscoveryRepository {
	return &AgentDiscoveryRepository{db: db}
}

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

	if r.db.Dialector.Name() == "mysql" {
		const chunk = 200
		for i := 0; i < len(items); i += chunk {
			end := i + chunk
			if end > len(items) {
				end = len(items)
			}
			batch := items[i:end]
			err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "project_id"},
					{Name: "server_id"},
					{Name: "kind"},
					{Name: "value"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"last_seen_at", "extra", "updated_at"}),
			}).Create(&batch).Error
			if err != nil {
				return r.upsertManyFallback(ctx, projectID, serverID, batch)
			}
		}
		return nil
	}
	return r.upsertManyFallback(ctx, projectID, serverID, items)
}

func (r *AgentDiscoveryRepository) upsertManyFallback(ctx context.Context, projectID, serverID uint, items []model.AgentDiscovery) error {
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

type AgentDiscoveryListFilter struct {
	ProjectID  uint
	ServerID   uint
	Kind       *string
	Limit      int
	Prefix     string
	FreshSince *time.Time
}

func (r *AgentDiscoveryRepository) List(ctx context.Context, f AgentDiscoveryListFilter) ([]model.AgentDiscovery, error) {
	limit := f.Limit
	if limit <= 0 || limit > 2000 {
		limit = 300
	}
	q := r.db.WithContext(ctx).Model(&model.AgentDiscovery{}).
		Where("project_id=? AND server_id=?", f.ProjectID, f.ServerID).
		Order("last_seen_at DESC").
		Limit(limit)
	if f.Kind != nil && *f.Kind != "" {
		q = q.Where("kind=?", *f.Kind)
	}
	if f.Prefix != "" {
		q = q.Where("value LIKE ?", f.Prefix+"%")
	}
	if f.FreshSince != nil {
		q = q.Where("last_seen_at >= ?", *f.FreshSince)
	}
	var out []model.AgentDiscovery
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// PruneStale removes discovery rows not seen since cutoff for a server.
func (r *AgentDiscoveryRepository) PruneStale(ctx context.Context, projectID, serverID uint, cutoff time.Time) error {
	return r.db.WithContext(ctx).
		Where("project_id=? AND server_id=? AND last_seen_at < ?", projectID, serverID, cutoff).
		Delete(&model.AgentDiscovery{}).Error
}
