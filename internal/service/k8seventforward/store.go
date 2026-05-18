package k8seventforward

import (
	"context"
	"errors"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveEvent(ctx context.Context, ev *model.K8sForwardedEvent) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "evt_key"}},
		DoNothing: true,
	}).Create(ev).Error
}

func (s *Store) ListUnprocessed(ctx context.Context, limit int) ([]model.K8sForwardedEvent, error) {
	var list []model.K8sForwardedEvent
	err := s.db.WithContext(ctx).
		Where("processed = ?", false).
		Order("timestamp ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (s *Store) MarkProcessed(ctx context.Context, id int64, processed bool) error {
	return s.db.WithContext(ctx).Model(&model.K8sForwardedEvent{}).
		Where("id = ?", id).
		Update("processed", processed).Error
}

func (s *Store) IncrementAttempts(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&model.K8sForwardedEvent{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + ?", 1)).Error
}

func (s *Store) ListEnabledRules(ctx context.Context) ([]model.K8sEventForwardRule, error) {
	var rules []model.K8sEventForwardRule
	err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

func (s *Store) HasEnabledRules(ctx context.Context) (bool, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&model.K8sEventForwardRule{}).
		Where("enabled = ?", true).Count(&n).Error
	return n > 0, err
}

func (s *Store) LoadSettings(ctx context.Context) (model.K8sEventForwardSetting, error) {
	var st model.K8sEventForwardSetting
	err := s.db.WithContext(ctx).First(&st, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.K8sEventForwardSetting{ID: 1}, nil
	}
	return st, err
}

func (s *Store) EnsureDefaultSettings(ctx context.Context, defaults model.K8sEventForwardSetting) error {
	var st model.K8sEventForwardSetting
	err := s.db.WithContext(ctx).First(&st, 1).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	defaults.ID = 1
	return s.db.WithContext(ctx).Create(&defaults).Error
}

func (s *Store) ListEnabledClusterIDs(ctx context.Context) ([]uint, error) {
	var ids []uint
	err := s.db.WithContext(ctx).Model(&model.K8sCluster{}).
		Where("status = ?", 1).
		Pluck("id", &ids).Error
	return ids, err
}

func (s *Store) GetClusterName(ctx context.Context, id uint) string {
	var c model.K8sCluster
	if err := s.db.WithContext(ctx).Select("name").First(&c, id).Error; err != nil {
		return ""
	}
	return c.Name
}

func nowUTC() time.Time { return time.Now().UTC() }
