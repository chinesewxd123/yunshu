package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type AlertReceiverGroupService struct {
	db    *gorm.DB
	cache *ReceiverGroupCache
}

func NewAlertReceiverGroupService(db *gorm.DB, cache *ReceiverGroupCache) *AlertReceiverGroupService {
	return &AlertReceiverGroupService{db: db, cache: cache}
}

type AlertReceiverGroupListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Enabled   *bool  `form:"enabled"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (s *AlertReceiverGroupService) List(ctx context.Context, q AlertReceiverGroupListQuery) ([]model.AlertReceiverGroup, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertReceiverGroup{})
	if q.ProjectID > 0 {
		tx = tx.Where("project_id = ?", q.ProjectID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if q.Enabled != nil {
		tx = tx.Where("enabled = ?", *q.Enabled)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []model.AlertReceiverGroup
	if err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	for i := range list {
		hydrateReceiverGroup(&list[i])
	}
	return list, total, page, pageSize, nil
}

type AlertReceiverGroupUpsertRequest struct {
	ProjectID uint    `json:"project_id" binding:"required"`
	Name      string  `json:"name" binding:"required,max=128"`
	Description string `json:"description"`

	ChannelIDsJSON       string `json:"channel_ids_json"`
	EmailRecipientsJSON  string `json:"email_recipients_json"`
	ActiveTimeStart      *string `json:"active_time_start"`
	ActiveTimeEnd        *string `json:"active_time_end"`
	WeekdaysJSON         string `json:"weekdays_json"`
	EscalationLevel      int    `json:"escalation_level"`
	Enabled              *bool  `json:"enabled"`
}

func (s *AlertReceiverGroupService) Create(ctx context.Context, req AlertReceiverGroupUpsertRequest) (*model.AlertReceiverGroup, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.BadRequest("name required")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.AlertReceiverGroup{
		ProjectID:            req.ProjectID,
		Name:                 strings.TrimSpace(req.Name),
		Description:          strings.TrimSpace(req.Description),
		ChannelIDsJSON:       strings.TrimSpace(req.ChannelIDsJSON),
		EmailRecipientsJSON:  strings.TrimSpace(req.EmailRecipientsJSON),
		ActiveTimeStart:      req.ActiveTimeStart,
		ActiveTimeEnd:        req.ActiveTimeEnd,
		WeekdaysJSON:         strings.TrimSpace(req.WeekdaysJSON),
		EscalationLevel:      req.EscalationLevel,
		Enabled:              enabled,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	hydrateReceiverGroup(row)
	return row, nil
}

func (s *AlertReceiverGroupService) Update(ctx context.Context, id uint, req AlertReceiverGroupUpsertRequest) (*model.AlertReceiverGroup, error) {
	var row model.AlertReceiverGroup
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, apperror.NotFound("receiver group not found")
	}
	if req.ProjectID > 0 && req.ProjectID != row.ProjectID {
		return nil, apperror.BadRequest("cannot change project_id")
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	row.Description = strings.TrimSpace(req.Description)
	row.ChannelIDsJSON = strings.TrimSpace(req.ChannelIDsJSON)
	row.EmailRecipientsJSON = strings.TrimSpace(req.EmailRecipientsJSON)
	row.ActiveTimeStart = req.ActiveTimeStart
	row.ActiveTimeEnd = req.ActiveTimeEnd
	row.WeekdaysJSON = strings.TrimSpace(req.WeekdaysJSON)
	row.EscalationLevel = req.EscalationLevel
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	hydrateReceiverGroup(&row)
	return &row, nil
}

func (s *AlertReceiverGroupService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertReceiverGroup{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("receiver group not found")
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	return nil
}

func hydrateReceiverGroup(it *model.AlertReceiverGroup) {
	if it == nil {
		return
	}
	it.ChannelIDs = parseUintSliceJSON(it.ChannelIDsJSON)
	it.EmailRecipients = parseStringSliceJSON(it.EmailRecipientsJSON)
	it.Weekdays = parseIntSliceJSON(it.WeekdaysJSON)
}

