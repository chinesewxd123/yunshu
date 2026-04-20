package repository

import (
	"context"
	"time"

	"go-permission-system/internal/model"

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

func (r *LogAgentRepository) TouchSeen(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.LogAgent{}).Where("id = ?", id).Updates(map[string]any{
		"last_seen_at": now,
		"status":       model.StatusEnabled,
	}).Error
}
