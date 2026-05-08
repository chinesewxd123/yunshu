package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogAgentRepository struct {
	db *gorm.DB
}

func NewLogAgentRepository(db *gorm.DB) *LogAgentRepository { return &LogAgentRepository{db: db} }

func (r *LogAgentRepository) GetByServerID(ctx context.Context, serverID uint) (*model.LogAgent, error) {
	var it model.LogAgent
	if err := r.db.WithContext(ctx).Where("server_id = ?", serverID).First(&it).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogAgentRepository) GetByProjectAndServer(ctx context.Context, projectID, serverID uint) (*model.LogAgent, error) {
	var it model.LogAgent
	if err := r.db.WithContext(ctx).Where("project_id = ? AND server_id = ?", projectID, serverID).First(&it).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogAgentRepository) ListByProject(ctx context.Context, projectID uint) ([]model.LogAgent, error) {
	var list []model.LogAgent
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *LogAgentRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.LogAgent, error) {
	var it model.LogAgent
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&it).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogAgentRepository) Create(ctx context.Context, it *model.LogAgent) error {
	return r.db.WithContext(ctx).Create(it).Error
}

func (r *LogAgentRepository) Save(ctx context.Context, it *model.LogAgent) error {
	return r.db.WithContext(ctx).Save(it).Error
}

func (r *LogAgentRepository) TouchSeen(ctx context.Context, id uint, heartbeatTimeout time.Duration) error {
	var agent model.LogAgent
	if err := r.db.WithContext(ctx).First(&agent, id).Error; err != nil {
		return err
	}
	now := time.Now()
	wasOffline := agent.LastSeenAt == nil || now.Sub(*agent.LastSeenAt) > heartbeatTimeout
	up := map[string]any{
		"last_seen_at": now,
		"status":       model.StatusEnabled,
	}
	if wasOffline {
		up["last_online_at"] = now
		up["offline_sweep_seen_at"] = nil
	}
	return r.db.WithContext(ctx).Model(&model.LogAgent{}).Where("id = ?", id).Updates(up).Error
}

// ListAll 扫描离线归因（全表 Agent，数据量按部署可控）。
func (r *LogAgentRepository) ListAll(ctx context.Context) ([]model.LogAgent, error) {
	var list []model.LogAgent
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateOfflineMarker 记录离线时刻与原因（定时扫描调用）。
func (r *LogAgentRepository) UpdateOfflineMarker(ctx context.Context, id uint, offlineAt time.Time, reason string, sweepSeen *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.LogAgent{}).Where("id = ?", id).Updates(map[string]any{
		"last_offline_at":          offlineAt,
		"last_offline_reason_code": reason,
		"offline_sweep_seen_at":    sweepSeen,
	}).Error
}
