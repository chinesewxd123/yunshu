package service

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type AlertMonitorRuleListQuery struct {
	DatasourceID *uint  `form:"datasource_id"`
	ProjectID    *uint  `form:"project_id"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type AlertMonitorRuleItem struct {
	model.AlertMonitorRule
	ProjectID     uint   `json:"project_id" gorm:"column:project_id"`
	ProjectName   string `json:"project_name,omitempty" gorm:"column:project_name"`
	DatasourceName string `json:"datasource_name,omitempty" gorm:"column:datasource_name"`
}

type AlertMonitorRuleUpsertRequest struct {
	DatasourceID        uint   `json:"datasource_id" binding:"required"`
	// ProjectID is deprecated: rule.project_id is derived from datasource.project_id.
	ProjectID           *uint  `json:"project_id"`
	Name                string `json:"name" binding:"required,max=128"`
	Expr                string `json:"expr" binding:"required"`
	ForSeconds          int    `json:"for_seconds"`
	EvalIntervalSeconds int    `json:"eval_interval_seconds"`
	Severity            string `json:"severity" binding:"omitempty,max=32"`
	ThresholdUnit       string `json:"threshold_unit" binding:"omitempty,max=32"`
	LabelsJSON          string `json:"labels_json"`
	AnnotationsJSON     string `json:"annotations_json"`
	Enabled             *bool  `json:"enabled"`
}

type AlertMonitorRuleService struct {
	db *gorm.DB
}

func NewAlertMonitorRuleService(db *gorm.DB) *AlertMonitorRuleService {
	return &AlertMonitorRuleService{db: db}
}

func (s *AlertMonitorRuleService) List(ctx context.Context, q AlertMonitorRuleListQuery) ([]AlertMonitorRuleItem, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Table("alert_monitor_rules amr").
		Select("amr.*, ad.project_id AS project_id, p.name AS project_name, ad.name AS datasource_name").
		Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
		Joins("LEFT JOIN projects p ON p.id = ad.project_id AND p.deleted_at IS NULL")
	if q.DatasourceID != nil && *q.DatasourceID > 0 {
		tx = tx.Where("amr.datasource_id = ?", *q.DatasourceID)
	}
	if q.ProjectID != nil {
		tx = tx.Where("ad.project_id = ?", *q.ProjectID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("amr.name LIKE ? OR amr.expr LIKE ? OR ad.name LIKE ? OR p.name LIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []AlertMonitorRuleItem
	if err := tx.Order("amr.id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	return list, total, page, pageSize, nil
}

func (s *AlertMonitorRuleService) ListEnabled(ctx context.Context) ([]AlertMonitorRuleItem, error) {
	var list []AlertMonitorRuleItem
	err := s.db.WithContext(ctx).
		Table("alert_monitor_rules amr").
		Select("amr.*, ad.project_id AS project_id").
		Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
		Where("amr.enabled = ?", true).
		Order("amr.id ASC").
		Find(&list).Error
	return list, err
}

func (s *AlertMonitorRuleService) Get(ctx context.Context, id uint) (*model.AlertMonitorRule, error) {
	var row model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("监控规则不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (s *AlertMonitorRuleService) Create(ctx context.Context, req AlertMonitorRuleUpsertRequest) (*model.AlertMonitorRule, error) {
	var ds model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&ds, req.DatasourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("数据源不存在")
		}
		return nil, err
	}
	if ds.ProjectID == 0 {
		return nil, apperror.BadRequest("数据源未绑定项目")
	}
	ev := req.EvalIntervalSeconds
	if ev <= 0 {
		ev = 30
	}
	if ev < 5 {
		ev = 5
	}
	sev := strings.TrimSpace(req.Severity)
	if sev == "" {
		sev = "warning"
	}
	unit := strings.TrimSpace(req.ThresholdUnit)
	if unit == "" {
		unit = "raw"
	}
	row := model.AlertMonitorRule{
		DatasourceID:        req.DatasourceID,
		Name:                strings.TrimSpace(req.Name),
		Expr:                strings.TrimSpace(req.Expr),
		ForSeconds:          req.ForSeconds,
		EvalIntervalSeconds: ev,
		Severity:            sev,
		ThresholdUnit:       unit,
		LabelsJSON:          strings.TrimSpace(req.LabelsJSON),
		AnnotationsJSON:     strings.TrimSpace(req.AnnotationsJSON),
		Enabled:             req.Enabled == nil || *req.Enabled,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertMonitorRuleService) Update(ctx context.Context, id uint, req AlertMonitorRuleUpsertRequest) (*model.AlertMonitorRule, error) {
	row, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// Always ensure datasource is valid. project_id is derived from datasource at read time.
	dsID := req.DatasourceID
	if dsID == 0 {
		dsID = row.DatasourceID
	}
	var ds model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&ds, dsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("数据源不存在")
		}
		return nil, err
	}
	if ds.ProjectID == 0 {
		return nil, apperror.BadRequest("数据源未绑定项目")
	}
	if req.DatasourceID > 0 && req.DatasourceID != row.DatasourceID {
		row.DatasourceID = req.DatasourceID
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Expr) != "" {
		row.Expr = strings.TrimSpace(req.Expr)
	}
	row.ForSeconds = req.ForSeconds
	if req.EvalIntervalSeconds > 0 {
		row.EvalIntervalSeconds = req.EvalIntervalSeconds
		if row.EvalIntervalSeconds < 5 {
			row.EvalIntervalSeconds = 5
		}
	}
	if strings.TrimSpace(req.Severity) != "" {
		row.Severity = strings.TrimSpace(req.Severity)
	}
	if strings.TrimSpace(req.ThresholdUnit) != "" {
		row.ThresholdUnit = strings.TrimSpace(req.ThresholdUnit)
	}
	row.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	row.AnnotationsJSON = strings.TrimSpace(req.AnnotationsJSON)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func uintOrZero(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func (s *AlertMonitorRuleService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("monitor_rule_id = ?", id).Delete(&model.AlertRuleAssignee{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&model.AlertMonitorRule{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperror.NotFound("监控规则不存在")
		}
		return nil
	})
}
