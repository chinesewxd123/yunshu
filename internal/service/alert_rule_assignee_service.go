package service

import (
	"context"
	"encoding/json"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"yunshu/internal/model"
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
	db             *gorm.DB
	userRepo       *repository.UserRepository
	memberRepo     *repository.ProjectMemberRepository
	departmentRepo *repository.DepartmentRepository
}

func NewAlertRuleAssigneeService(db *gorm.DB, userRepo *repository.UserRepository, memberRepo *repository.ProjectMemberRepository, departmentRepo *repository.DepartmentRepository) *AlertRuleAssigneeService {
	return &AlertRuleAssigneeService{db: db, userRepo: userRepo, memberRepo: memberRepo, departmentRepo: departmentRepo}
}

func assigneeParseStringSliceJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var ss []string
	if err := json.Unmarshal([]byte(raw), &ss); err != nil {
		return nil, svcerr.Pass(context.Background(), "alert.assignee", "assigneeParseStringSliceJSON", err)
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
	return list, svcerr.Pass(ctx, "alert.assignee", "ListByRule", err)
}

func (s *AlertRuleAssigneeService) UpsertPrimary(ctx context.Context, ruleID uint, req AlertRuleAssigneeUpsertRequest) (*model.AlertRuleAssignee, error) {
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, ruleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, svcerr.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
	}
	var row model.AlertRuleAssignee
	err := s.db.WithContext(ctx).Where("monitor_rule_id = ?", ruleID).First(&row).Error
	isNew := err == gorm.ErrRecordNotFound
	if err != nil && !isNew {
		return nil, svcerr.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
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
			return nil, svcerr.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
		}
		return &row, nil
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
	}
	return &row, nil
}

func (s *AlertRuleAssigneeService) resolveRuleProjectID(ctx context.Context, ruleID uint) (uint, error) {
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, ruleID).Error; err != nil {
		return 0, svcerr.Pass(ctx, "alert.assignee", "resolveRuleProjectID", err)
	}
	var ds model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&ds, rule.DatasourceID).Error; err != nil {
		return 0, svcerr.Pass(ctx, "alert.assignee", "resolveRuleProjectID", err)
	}
	return ds.ProjectID, nil
}

func (s *AlertRuleAssigneeService) expandDepartmentProjectMemberUserIDs(ctx context.Context, projectID uint, deptRootIDs []uint) ([]uint, error) {
	if projectID == 0 || len(deptRootIDs) == 0 || s.memberRepo == nil || s.departmentRepo == nil {
		return nil, nil
	}
	seen := map[uint]struct{}{}
	var deptIDs []uint
	for _, root := range deptRootIDs {
		if root == 0 {
			continue
		}
		sub, err := s.departmentRepo.ListDescendantIDsAndSelf(ctx, root)
		if err != nil {
			return nil, svcerr.Pass(ctx, "alert.assignee", "expandDepartmentProjectMemberUserIDs", err)
		}
		for _, id := range sub {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			deptIDs = append(deptIDs, id)
		}
	}
	if len(deptIDs) == 0 {
		return nil, nil
	}
	var uids []uint
	err := s.db.WithContext(ctx).Table("project_members AS pm").
		Select("DISTINCT pm.user_id").
		Joins("INNER JOIN users u ON u.id = pm.user_id AND u.deleted_at IS NULL").
		Where("pm.project_id = ? AND pm.deleted_at IS NULL AND u.department_id IN ?", projectID, deptIDs).
		Pluck("pm.user_id", &uids).Error
	return uids, svcerr.Pass(ctx, "alert.assignee", "expandDepartmentProjectMemberUserIDs", err)
}

func (s *AlertRuleAssigneeService) leaderUserIDsFromDepartmentRoots(ctx context.Context, deptRootIDs []uint) ([]uint, error) {
	if s.departmentRepo == nil || len(deptRootIDs) == 0 {
		return nil, nil
	}
	seen := map[uint]struct{}{}
	var out []uint
	for _, root := range deptRootIDs {
		if root == 0 {
			continue
		}
		d, err := s.departmentRepo.GetByID(ctx, root)
		if err != nil || d == nil || d.LeaderID == nil || *d.LeaderID == 0 {
			continue
		}
		if _, ok := seen[*d.LeaderID]; ok {
			continue
		}
		seen[*d.LeaderID] = struct{}{}
		out = append(out, *d.LeaderID)
	}
	return out, nil
}

// NotifyOnResolvedEnabled 规则是否配置「恢复时通知处理人」。
func (s *AlertRuleAssigneeService) NotifyOnResolvedEnabled(ctx context.Context, ruleID uint) bool {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return false
	}
	for _, row := range list {
		if row.NotifyOnResolved {
			return true
		}
	}
	return false
}

// ResolveNotifyEmailsDirectUsers 仅解析规则处理人中的「显式用户」与「额外邮箱」，不展开部门子树（用于 critical 邮件兜底，避免误发项目内非处理人）。
func (s *AlertRuleAssigneeService) ResolveNotifyEmailsDirectUsers(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyEmailsDirectUsers", err)
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
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
	}
	var allUIDs []uint
	for id := range uidSet {
		allUIDs = append(allUIDs, id)
	}
	if len(allUIDs) > 0 {
		users, err := s.userRepo.ListByIDs(ctx, allUIDs)
		if err != nil {
			return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyEmailsDirectUsers", err)
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
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ResolveNotifyEmails 合并规则处理人邮箱：显式用户 + 所选部门在「规则所属项目」内的成员 + 所选根部门的负责人 + 额外邮箱。
func (s *AlertRuleAssigneeService) ResolveNotifyEmails(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
	}
	projectID, err := s.resolveRuleProjectID(ctx, ruleID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
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
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
		deptRoots := parseUintSliceJSON(row.DepartmentIDsJSON)
		more, e1 := s.expandDepartmentProjectMemberUserIDs(ctx, projectID, deptRoots)
		if e1 != nil {
			return nil, e1
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		leaders, e2 := s.leaderUserIDsFromDepartmentRoots(ctx, deptRoots)
		if e2 != nil {
			return nil, e2
		}
		for _, id := range leaders {
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
			return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
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
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ResolveNotifyPhones 合并规则处理人手机号：显式用户 + 部门在项目内的成员 + 根部门负责人。
func (s *AlertRuleAssigneeService) ResolveNotifyPhones(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
	}
	projectID, err := s.resolveRuleProjectID(ctx, ruleID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
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
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
		deptRoots := parseUintSliceJSON(row.DepartmentIDsJSON)
		more, e1 := s.expandDepartmentProjectMemberUserIDs(ctx, projectID, deptRoots)
		if e1 != nil {
			return nil, e1
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		leaders, e2 := s.leaderUserIDsFromDepartmentRoots(ctx, deptRoots)
		if e2 != nil {
			return nil, e2
		}
		for _, id := range leaders {
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
			return nil, svcerr.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
		}
		for i := range users {
			add(users[i].Phone)
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
		return constants.ErrNotFoundWithMsg(constants.ErrMsg8faff6dbdd1d)
	}
	return nil
}

func marshalUintSliceJSON(ids []uint) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// PruneUserFromAllAssignees 从所有监控规则处理人配置中移除已删除用户 ID。
func (s *AlertRuleAssigneeService) PruneUserFromAllAssignees(ctx context.Context, userID uint) error {
	if userID == 0 || s.db == nil {
		return nil
	}
	var rows []model.AlertRuleAssignee
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return svcerr.Pass(ctx, "alert.assignee", "PruneUserFromAllAssignees", err)
	}
	for i := range rows {
		ids := parseUintSliceJSON(rows[i].UserIDsJSON)
		if len(ids) == 0 {
			continue
		}
		filtered := make([]uint, 0, len(ids))
		for _, id := range ids {
			if id != userID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == len(ids) {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&model.AlertRuleAssignee{}).Where("id = ?", rows[i].ID).
			Update("user_ids_json", marshalUintSliceJSON(filtered)).Error; err != nil {
			return svcerr.Pass(ctx, "alert.assignee", "PruneUserFromAllAssignees", err)
		}
	}
	return nil
}

// PruneDepartmentFromAllAssignees 从所有监控规则处理人配置中移除已删除部门 ID。
func (s *AlertRuleAssigneeService) PruneDepartmentFromAllAssignees(ctx context.Context, departmentID uint) error {
	if departmentID == 0 || s.db == nil {
		return nil
	}
	var rows []model.AlertRuleAssignee
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return svcerr.Pass(ctx, "alert.assignee", "PruneDepartmentFromAllAssignees", err)
	}
	for i := range rows {
		ids := parseUintSliceJSON(rows[i].DepartmentIDsJSON)
		if len(ids) == 0 {
			continue
		}
		filtered := make([]uint, 0, len(ids))
		for _, id := range ids {
			if id != departmentID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == len(ids) {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&model.AlertRuleAssignee{}).Where("id = ?", rows[i].ID).
			Update("department_ids_json", marshalUintSliceJSON(filtered)).Error; err != nil {
			return svcerr.Pass(ctx, "alert.assignee", "PruneDepartmentFromAllAssignees", err)
		}
	}
	return nil
}
