package service

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"

	"gorm.io/gorm"
)

// SilenceMatcher 单条 matcher，语义参考 Alertmanager。
type SilenceMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"is_regex"`
}

type AlertSilenceListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type AlertSilenceUpsertRequest struct {
	Name         string    `json:"name" binding:"required,max=128"`
	MatchersJSON string    `json:"matchers_json" binding:"required"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	Comment      string    `json:"comment" binding:"omitempty,max=512"`
	Enabled      *bool     `json:"enabled"`
}

type AlertSilenceBatchItem struct {
	Name         string    `json:"name" binding:"required,max=128"`
	MatchersJSON string    `json:"matchers_json" binding:"required"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	Comment      string    `json:"comment" binding:"omitempty,max=512"`
	Enabled      *bool     `json:"enabled"`
}

type AlertSilenceBatchRequest struct {
	Items []AlertSilenceBatchItem `json:"items" binding:"required,min=1"`
}

type AlertSilenceService struct {
	db *gorm.DB
}

func NewAlertSilenceService(db *gorm.DB) *AlertSilenceService {
	return &AlertSilenceService{db: db}
}

// disableExpiredSilences 将已过期但仍启用的静默自动停用，避免 UI 显示“启用”造成误解。
// 这是轻量级的“读时修正”：不引入定时任务，也不影响未过期静默的正常流程。
func (s *AlertSilenceService) disableExpiredSilences(ctx context.Context, now time.Time) {
	if s == nil || s.db == nil {
		return
	}
	// best-effort：失败不影响主流程
	_ = s.db.WithContext(ctx).
		Model(&model.AlertSilence{}).
		Where("enabled = ? AND ends_at < ?", true, now).
		Update("enabled", false).Error
}

func ParseSilenceMatchersJSON(raw string) ([]SilenceMatcher, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var ms []SilenceMatcher
	if err := json.Unmarshal([]byte(raw), &ms); err != nil {
		return nil, err
	}
	for _, m := range ms {
		if strings.TrimSpace(m.Name) == "" {
			return nil, apperror.BadRequest("matcher name 不能为空")
		}
	}
	return ms, nil
}

func normalizeSilenceMatchersKey(ms []SilenceMatcher) string {
	if len(ms) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		name := strings.TrimSpace(m.Name)
		value := strings.TrimSpace(m.Value)
		regex := "0"
		if m.IsRegex {
			regex = "1"
		}
		parts = append(parts, name+"|"+value+"|"+regex)
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func (s *AlertSilenceService) hasEnabledUnexpiredDuplicate(ctx context.Context, matchersJSON string, now time.Time) (bool, error) {
	targetMatchers, err := ParseSilenceMatchersJSON(matchersJSON)
	if err != nil {
		return false, err
	}
	targetKey := normalizeSilenceMatchersKey(targetMatchers)
	if targetKey == "" {
		return false, nil
	}
	var list []model.AlertSilence
	if err := s.db.WithContext(ctx).
		Where("enabled = ? AND ends_at > ?", true, now).
		Find(&list).Error; err != nil {
		return false, err
	}
	for _, row := range list {
		ms, err := ParseSilenceMatchersJSON(row.MatchersJSON)
		if err != nil {
			continue
		}
		if normalizeSilenceMatchersKey(ms) == targetKey {
			return true, nil
		}
	}
	return false, nil
}

// LabelsMatchSilenceMatchers 全部 matcher 命中 labels 时返回 true；matchers 为空视为匹配全部。
func LabelsMatchSilenceMatchers(ms []SilenceMatcher, labels map[string]string) bool {
	if len(ms) == 0 {
		return true
	}
	get := func(k string) string {
		if labels == nil {
			return ""
		}
		return strings.TrimSpace(labels[k])
	}
	for _, m := range ms {
		name := strings.TrimSpace(m.Name)
		want := m.Value
		got := get(name)
		if m.IsRegex {
			re, err := regexp.Compile("^(?:" + want + ")$")
			if err != nil || !re.MatchString(got) {
				return false
			}
		} else {
			if got != strings.TrimSpace(want) {
				return false
			}
		}
	}
	return true
}

func (s *AlertSilenceService) List(ctx context.Context, q AlertSilenceListQuery) ([]model.AlertSilence, int64, int, int, error) {
	s.disableExpiredSilences(ctx, time.Now())
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertSilence{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR comment LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []model.AlertSilence
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	return list, total, page, pageSize, nil
}

// ListActiveAt 返回在 t 时刻生效的静默（用于 Webhook 入口）。
func (s *AlertSilenceService) ListActiveAt(ctx context.Context, t time.Time) ([]model.AlertSilence, error) {
	s.disableExpiredSilences(ctx, t)
	var list []model.AlertSilence
	err := s.db.WithContext(ctx).Model(&model.AlertSilence{}).
		Where("enabled = ? AND starts_at <= ? AND ends_at >= ?", true, t, t).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (s *AlertSilenceService) FirstMatchingSilenceID(ctx context.Context, labels map[string]string, t time.Time) (uint, bool, error) {
	list, err := s.ListActiveAt(ctx, t)
	if err != nil {
		return 0, false, err
	}
	for _, sil := range list {
		ms, err := ParseSilenceMatchersJSON(sil.MatchersJSON)
		if err != nil {
			continue
		}
		if LabelsMatchSilenceMatchers(ms, labels) {
			return sil.ID, true, nil
		}
	}
	return 0, false, nil
}

func (s *AlertSilenceService) Create(ctx context.Context, userID uint, req AlertSilenceUpsertRequest) (*model.AlertSilence, error) {
	if _, err := ParseSilenceMatchersJSON(req.MatchersJSON); err != nil {
		return nil, err
	}
	if !req.EndsAt.After(req.StartsAt) {
		return nil, apperror.BadRequest("ends_at 必须晚于 starts_at")
	}
	dup, err := s.hasEnabledUnexpiredDuplicate(ctx, req.MatchersJSON, time.Now())
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, apperror.BadRequest("已存在启用且未过期的同类静默规则，无需重复创建；如需调整请编辑现有静默")
	}
	row := model.AlertSilence{
		Name:         strings.TrimSpace(req.Name),
		MatchersJSON: strings.TrimSpace(req.MatchersJSON),
		StartsAt:     req.StartsAt,
		EndsAt:       req.EndsAt,
		Comment:      strings.TrimSpace(req.Comment),
		CreatedBy:    userID,
		Enabled:      req.Enabled == nil || *req.Enabled,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertSilenceService) Update(ctx context.Context, id uint, req AlertSilenceUpsertRequest) (*model.AlertSilence, error) {
	var row model.AlertSilence
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("静默不存在")
		}
		return nil, err
	}
	if strings.TrimSpace(req.MatchersJSON) != "" {
		if _, err := ParseSilenceMatchersJSON(req.MatchersJSON); err != nil {
			return nil, err
		}
		row.MatchersJSON = strings.TrimSpace(req.MatchersJSON)
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if !req.StartsAt.IsZero() {
		row.StartsAt = req.StartsAt
	}
	if !req.EndsAt.IsZero() {
		row.EndsAt = req.EndsAt
	}
	if row.EndsAt.Before(row.StartsAt) || row.EndsAt.Equal(row.StartsAt) {
		return nil, apperror.BadRequest("ends_at 必须晚于 starts_at")
	}
	row.Comment = strings.TrimSpace(req.Comment)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertSilenceService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertSilence{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("静默不存在")
	}
	return nil
}

func (s *AlertSilenceService) CreateBatch(ctx context.Context, userID uint, req AlertSilenceBatchRequest) (int, error) {
	n := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			if _, err := ParseSilenceMatchersJSON(it.MatchersJSON); err != nil {
				return err
			}
			if !it.EndsAt.After(it.StartsAt) {
				return apperror.BadRequest("批量项 ends_at 必须晚于 starts_at: " + it.Name)
			}
			dup, err := s.hasEnabledUnexpiredDuplicate(ctx, it.MatchersJSON, time.Now())
			if err != nil {
				return err
			}
			if dup {
				return apperror.BadRequest("已存在启用且未过期的同类静默规则，请勿重复创建（批量任务已中止）")
			}
			row := model.AlertSilence{
				Name:         strings.TrimSpace(it.Name),
				MatchersJSON: strings.TrimSpace(it.MatchersJSON),
				StartsAt:     it.StartsAt,
				EndsAt:       it.EndsAt,
				Comment:      strings.TrimSpace(it.Comment),
				CreatedBy:    userID,
				Enabled:      it.Enabled == nil || *it.Enabled,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}
