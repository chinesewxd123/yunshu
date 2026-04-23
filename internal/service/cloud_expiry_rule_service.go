package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type CloudExpiryRuleListQuery struct {
	ProjectID *uint  `form:"project_id"`
	Provider  string `form:"provider"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type CloudExpiryRuleUpsertRequest struct {
	ProjectID           uint   `json:"project_id" binding:"required"`
	Name                string `json:"name" binding:"required,max=128"`
	Provider            string `json:"provider"`
	RegionScope         string `json:"region_scope"`
	AdvanceDays         int    `json:"advance_days"`
	Severity            string `json:"severity" binding:"omitempty,max=32"`
	LabelsJSON          string `json:"labels_json"`
	EvalIntervalSeconds int    `json:"eval_interval_seconds"`
	Enabled             *bool  `json:"enabled"`
}

type CloudExpiryRuleService struct {
	db *gorm.DB
}

func NewCloudExpiryRuleService(db *gorm.DB) *CloudExpiryRuleService {
	return &CloudExpiryRuleService{db: db}
}

func (s *CloudExpiryRuleService) List(ctx context.Context, q CloudExpiryRuleListQuery) ([]model.CloudExpiryRule, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.CloudExpiryRule{})
	if q.ProjectID != nil && *q.ProjectID > 0 {
		tx = tx.Where("project_id = ?", *q.ProjectID)
	}
	if p := strings.TrimSpace(q.Provider); p != "" {
		tx = tx.Where("provider = ?", p)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR region_scope LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []model.CloudExpiryRule
	if err := tx.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	return list, total, page, pageSize, nil
}

func (s *CloudExpiryRuleService) Create(ctx context.Context, req CloudExpiryRuleUpsertRequest) (*model.CloudExpiryRule, error) {
	ev := req.EvalIntervalSeconds
	if ev <= 0 {
		ev = 3600
	}
	if ev < 60 {
		ev = 60
	}
	days := req.AdvanceDays
	if days <= 0 {
		days = 7
	}
	sev := strings.TrimSpace(req.Severity)
	if sev == "" {
		sev = "warning"
	}
	row := model.CloudExpiryRule{
		ProjectID:           req.ProjectID,
		Name:                strings.TrimSpace(req.Name),
		Provider:            strings.TrimSpace(req.Provider),
		RegionScope:         strings.TrimSpace(req.RegionScope),
		AdvanceDays:         days,
		Severity:            sev,
		LabelsJSON:          strings.TrimSpace(req.LabelsJSON),
		EvalIntervalSeconds: ev,
		Enabled:             req.Enabled == nil || *req.Enabled,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *CloudExpiryRuleService) Update(ctx context.Context, id uint, req CloudExpiryRuleUpsertRequest) (*model.CloudExpiryRule, error) {
	var row model.CloudExpiryRule
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("云到期规则不存在")
		}
		return nil, err
	}
	if req.ProjectID > 0 {
		row.ProjectID = req.ProjectID
	}
	if v := strings.TrimSpace(req.Name); v != "" {
		row.Name = v
	}
	row.Provider = strings.TrimSpace(req.Provider)
	row.RegionScope = strings.TrimSpace(req.RegionScope)
	if req.AdvanceDays > 0 {
		row.AdvanceDays = req.AdvanceDays
	}
	if v := strings.TrimSpace(req.Severity); v != "" {
		row.Severity = v
	}
	row.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	if req.EvalIntervalSeconds > 0 {
		row.EvalIntervalSeconds = req.EvalIntervalSeconds
		if row.EvalIntervalSeconds < 60 {
			row.EvalIntervalSeconds = 60
		}
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *CloudExpiryRuleService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.CloudExpiryRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("云到期规则不存在")
	}
	return nil
}
