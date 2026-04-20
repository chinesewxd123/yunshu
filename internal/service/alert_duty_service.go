package service

import (
	"context"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/repository"

	"gorm.io/gorm"
)

type AlertDutyBlockListQuery struct {
	MonitorRuleID *uint `form:"monitor_rule_id"`
	ProjectID     *uint `form:"project_id"`
	Page          int   `form:"page"`
	PageSize      int   `form:"page_size"`
}

type AlertDutyBlockUpsertRequest struct {
	MonitorRuleID     uint      `json:"monitor_rule_id" binding:"required"`
	StartsAt            time.Time `json:"starts_at" binding:"required"`
	EndsAt              time.Time `json:"ends_at" binding:"required"`
	Title               string    `json:"title" binding:"omitempty,max=128"`
	UserIDsJSON         string    `json:"user_ids_json"`
	DepartmentIDsJSON   string    `json:"department_ids_json"`
	ExtraEmailsJSON     string    `json:"extra_emails_json"`
	Remark              string    `json:"remark" binding:"omitempty,max=512"`
}

type AlertDutyService struct {
	db       *gorm.DB
	userRepo *repository.UserRepository
}

func NewAlertDutyService(db *gorm.DB, userRepo *repository.UserRepository) *AlertDutyService {
	return &AlertDutyService{db: db, userRepo: userRepo}
}

func (s *AlertDutyService) ListBlocks(ctx context.Context, q AlertDutyBlockListQuery) ([]model.AlertDutyBlock, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertDutyBlock{})
	if q.MonitorRuleID != nil && *q.MonitorRuleID > 0 {
		tx = tx.Where("monitor_rule_id = ?", *q.MonitorRuleID)
	}
	if q.ProjectID != nil {
		tx = tx.Joins("JOIN alert_monitor_rules amr ON amr.id = alert_duty_blocks.monitor_rule_id").Where("amr.project_id = ?", *q.ProjectID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []model.AlertDutyBlock
	if err := tx.Order("starts_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDutyService) CreateBlock(ctx context.Context, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	if !req.EndsAt.After(req.StartsAt) {
		return nil, apperror.BadRequest("ends_at 必须晚于 starts_at")
	}
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, req.MonitorRuleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("监控规则不存在")
		}
		return nil, err
	}
	row := model.AlertDutyBlock{
		MonitorRuleID:     req.MonitorRuleID,
		StartsAt:          req.StartsAt,
		EndsAt:            req.EndsAt,
		Title:             strings.TrimSpace(req.Title),
		UserIDsJSON:       strings.TrimSpace(req.UserIDsJSON),
		DepartmentIDsJSON: strings.TrimSpace(req.DepartmentIDsJSON),
		ExtraEmailsJSON:   strings.TrimSpace(req.ExtraEmailsJSON),
		Remark:            strings.TrimSpace(req.Remark),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertDutyService) UpdateBlock(ctx context.Context, id uint, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	var row model.AlertDutyBlock
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("班次块不存在")
		}
		return nil, err
	}
	if req.MonitorRuleID > 0 && req.MonitorRuleID != row.MonitorRuleID {
		var rule model.AlertMonitorRule
		if err := s.db.WithContext(ctx).First(&rule, req.MonitorRuleID).Error; err != nil {
			return nil, apperror.BadRequest("监控规则不存在")
		}
		row.MonitorRuleID = req.MonitorRuleID
	}
	if !req.StartsAt.IsZero() {
		row.StartsAt = req.StartsAt
	}
	if !req.EndsAt.IsZero() {
		row.EndsAt = req.EndsAt
	}
	if !row.EndsAt.After(row.StartsAt) {
		return nil, apperror.BadRequest("ends_at 必须晚于 starts_at")
	}
	row.Title = strings.TrimSpace(req.Title)
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	row.DepartmentIDsJSON = strings.TrimSpace(req.DepartmentIDsJSON)
	row.ExtraEmailsJSON = strings.TrimSpace(req.ExtraEmailsJSON)
	row.Remark = strings.TrimSpace(req.Remark)
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertDutyService) DeleteBlock(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertDutyBlock{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("班次块不存在")
	}
	return nil
}

func dedupeEmailsLower(emails []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

// ResolveNotifyEmailsAtRule 合并当前时刻命中的班次块内的用户/部门子树/额外邮箱（直接绑定到监控规则）。
func (s *AlertDutyService) ResolveNotifyEmailsAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]string, error) {
	var blocks []model.AlertDutyBlock
	if err := s.db.WithContext(ctx).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Order("id ASC").
		Find(&blocks).Error; err != nil {
		return nil, err
	}
	var emails []string
	for _, b := range blocks {
		uidSet := map[uint]struct{}{}
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			uidSet[id] = struct{}{}
		}
		deptRoots := parseUintSliceJSON(b.DepartmentIDsJSON)
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptRoots)
		if err != nil {
			return nil, err
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		var all []uint
		for id := range uidSet {
			all = append(all, id)
		}
		if len(all) > 0 {
			users, err := s.userRepo.ListByIDs(ctx, all)
			if err != nil {
				return nil, err
			}
			for i := range users {
				if users[i].Email != nil {
					emails = append(emails, *users[i].Email)
				}
			}
		}
		extras, _ := assigneeParseStringSliceJSON(b.ExtraEmailsJSON)
		emails = append(emails, extras...)
	}
	return dedupeEmailsLower(emails), nil
}
