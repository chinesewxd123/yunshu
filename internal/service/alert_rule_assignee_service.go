package service

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type AlertRuleAssigneeUpsertRequest struct {
	UserIDsJSON       string `json:"user_ids_json"`
	DepartmentIDsJSON string `json:"department_ids_json"`
	ExtraEmailsJSON   string `json:"extra_emails_json"`
	NotifyOnResolved  *bool  `json:"notify_on_resolved"`
	Remark            string `json:"remark" binding:"omitempty,max=512"`
}

type AlertRuleAssigneeService struct {
	db         *gorm.DB
	userRepo   *repository.UserRepository
	memberRepo *repository.ProjectMemberRepository
}

func NewAlertRuleAssigneeService(db *gorm.DB, userRepo *repository.UserRepository, memberRepo *repository.ProjectMemberRepository) *AlertRuleAssigneeService {
	return &AlertRuleAssigneeService{db: db, userRepo: userRepo, memberRepo: memberRepo}
}

func assigneeParseStringSliceJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var ss []string
	if err := json.Unmarshal([]byte(raw), &ss); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func (s *AlertRuleAssigneeService) ListByRule(ctx context.Context, ruleID uint) ([]model.AlertRuleAssignee, error) {
	var list []model.AlertRuleAssignee
	err := s.db.WithContext(ctx).Where("monitor_rule_id = ?", ruleID).Order("id ASC").Find(&list).Error
	return list, err
}

func (s *AlertRuleAssigneeService) UpsertPrimary(ctx context.Context, ruleID uint, req AlertRuleAssigneeUpsertRequest) (*model.AlertRuleAssignee, error) {
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, ruleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("监控规则不存在")
		}
		return nil, err
	}
	var row model.AlertRuleAssignee
	err := s.db.WithContext(ctx).Where("monitor_rule_id = ?", ruleID).First(&row).Error
	isNew := err == gorm.ErrRecordNotFound
	if err != nil && !isNew {
		return nil, err
	}
	row.MonitorRuleID = ruleID
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	row.DepartmentIDsJSON = strings.TrimSpace(req.DepartmentIDsJSON)
	row.ExtraEmailsJSON = strings.TrimSpace(req.ExtraEmailsJSON)
	row.Remark = strings.TrimSpace(req.Remark)
	if req.NotifyOnResolved != nil {
		row.NotifyOnResolved = *req.NotifyOnResolved
	}
	if isNew {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ResolveNotifyEmails 合并：规则处理人（用户/部门/extra）+ 项目成员邮箱（规则绑定 project_id 时）。
func (s *AlertRuleAssigneeService) ResolveNotifyEmails(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	if len(list) > 0 && s.userRepo != nil {
		uidSet := map[uint]struct{}{}
		for _, row := range list {
			for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
				uidSet[id] = struct{}{}
			}
			deptIDs := parseUintSliceJSON(row.DepartmentIDsJSON)
			moreIDs, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptIDs)
			if err != nil {
				return nil, err
			}
			for _, id := range moreIDs {
				uidSet[id] = struct{}{}
			}
		}
		var allUIDs []uint
		for id := range uidSet {
			allUIDs = append(allUIDs, id)
		}
		if len(allUIDs) > 0 {
			users, err := s.userRepo.ListByIDs(ctx, allUIDs)
			if err != nil {
				return nil, err
			}
			for i := range users {
				if users[i].Email != nil {
					add(*users[i].Email)
				}
			}
		}
		for _, row := range list {
			extras, _ := assigneeParseStringSliceJSON(row.ExtraEmailsJSON)
			for _, e := range extras {
				add(e)
			}
		}
	}
	// 未配置处理人时仍可仅依赖项目成员；与 project_members 表对齐。
	if s.memberRepo != nil && s.userRepo != nil {
		var projectID uint
		_ = s.db.WithContext(ctx).
			Table("alert_monitor_rules amr").
			Select("ad.project_id AS project_id").
			Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
			Where("amr.id = ? AND amr.deleted_at IS NULL", ruleID).
			Scan(&projectID).Error
		if projectID > 0 {
			uids, err := s.memberRepo.ListUserIDsByProject(ctx, projectID)
			if err == nil && len(uids) > 0 {
				users, err := s.userRepo.ListByIDs(ctx, uids)
				if err == nil {
					for i := range users {
						if users[i].Status == model.StatusEnabled && users[i].Email != nil {
							add(*users[i].Email)
						}
					}
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *AlertRuleAssigneeService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertRuleAssignee{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("处理人配置不存在")
	}
	return nil
}
