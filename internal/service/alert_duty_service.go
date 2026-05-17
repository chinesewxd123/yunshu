package service

import (
	"context"
	"strings"
	"time"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

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
	StartsAt          time.Time `json:"starts_at" binding:"required"`
	EndsAt            time.Time `json:"ends_at" binding:"required"`
	Title             string    `json:"title" binding:"omitempty,max=128"`
	UserIDsJSON       string    `json:"user_ids_json"`
	DepartmentIDsJSON string    `json:"department_ids_json"`
	ExtraEmailsJSON   string    `json:"extra_emails_json"`
	Remark            string    `json:"remark" binding:"omitempty,max=512"`
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
		tx = tx.
			Joins("JOIN alert_monitor_rules amr ON amr.id = alert_duty_blocks.monitor_rule_id AND amr.deleted_at IS NULL").
			Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
			Where("ad.project_id = ?", *q.ProjectID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, svcerr.Pass("alert.duty", "ListBlocks", err)
	}
	var list []model.AlertDutyBlock
	if err := tx.Order("starts_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, svcerr.Pass("alert.duty", "ListBlocks", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDutyService) CreateBlock(ctx context.Context, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	if !req.EndsAt.After(req.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, req.MonitorRuleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, svcerr.Pass("alert.duty", "CreateBlock", err)
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
		return nil, svcerr.Pass("alert.duty", "CreateBlock", err)
	}
	return &row, nil
}

func (s *AlertDutyService) UpdateBlock(ctx context.Context, id uint, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	var row model.AlertDutyBlock
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
		}
		return nil, svcerr.Pass("alert.duty", "UpdateBlock", err)
	}
	if req.MonitorRuleID > 0 && req.MonitorRuleID != row.MonitorRuleID {
		var rule model.AlertMonitorRule
		if err := s.db.WithContext(ctx).First(&rule, req.MonitorRuleID).Error; err != nil {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
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
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	row.Title = strings.TrimSpace(req.Title)
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	row.DepartmentIDsJSON = strings.TrimSpace(req.DepartmentIDsJSON)
	row.ExtraEmailsJSON = strings.TrimSpace(req.ExtraEmailsJSON)
	row.Remark = strings.TrimSpace(req.Remark)
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, svcerr.Pass("alert.duty", "UpdateBlock", err)
	}
	return &row, nil
}

func (s *AlertDutyService) DeleteBlock(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertDutyBlock{}, id)
	if res.Error != nil {
		return svcerr.Pass("alert.duty", "DeleteBlock", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
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

// HasActiveBlockAtRule 判断指定时刻是否存在覆盖该监控规则的值班班次。
func (s *AlertDutyService) HasActiveBlockAtRule(ctx context.Context, monitorRuleID uint, t time.Time) (bool, error) {
	if monitorRuleID == 0 {
		return false, nil
	}
	var n int64
	err := s.db.WithContext(ctx).Model(&model.AlertDutyBlock{}).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Limit(1).
		Count(&n).Error
	return n > 0, svcerr.Pass("alert.duty", "HasActiveBlockAtRule", err)
}

// ResolveNotifyEmailsAtRule 合并当前时刻命中的班次块内的用户/部门子树/额外邮箱（直接绑定到监控规则）。
func (s *AlertDutyService) ResolveNotifyEmailsAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]string, error) {
	var blocks []model.AlertDutyBlock
	if err := s.db.WithContext(ctx).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Order("id ASC").
		Find(&blocks).Error; err != nil {
		return nil, svcerr.Pass("alert.duty", "ResolveNotifyEmailsAtRule", err)
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
			return nil, svcerr.Pass("alert.duty", "ResolveNotifyEmailsAtRule", err)
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
				return nil, svcerr.Pass("alert.duty", "ResolveNotifyEmailsAtRule", err)
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

// ResolveNotifyPhonesAtRule 合并当前时刻命中班次块内用户手机号（含部门子树用户，不含 extra 邮箱侧电话）。
func (s *AlertDutyService) ResolveNotifyPhonesAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]string, error) {
	var blocks []model.AlertDutyBlock
	if err := s.db.WithContext(ctx).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Order("id ASC").
		Find(&blocks).Error; err != nil {
		return nil, svcerr.Pass("alert.duty", "ResolveNotifyPhonesAtRule", err)
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, b := range blocks {
		uidSet := map[uint]struct{}{}
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			uidSet[id] = struct{}{}
		}
		deptRoots := parseUintSliceJSON(b.DepartmentIDsJSON)
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptRoots)
		if err != nil {
			return nil, svcerr.Pass("alert.duty", "ResolveNotifyPhonesAtRule", err)
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
				return nil, svcerr.Pass("alert.duty", "ResolveNotifyPhonesAtRule", err)
			}
			for i := range users {
				add(users[i].Phone)
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
