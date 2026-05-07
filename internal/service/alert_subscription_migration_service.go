package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
)

type SubscriptionMigrationReport struct {
	PoliciesTotal         int `json:"policies_total"`
	PoliciesMigrated      int `json:"policies_migrated"`
	ReceiverGroupsCreated int `json:"receiver_groups_created"`
	NodesCreated          int `json:"nodes_created"`
	PoliciesDisabled      int `json:"policies_disabled"`
	// ResolvedDefaultProjectID 当策略 match_labels 未带 project_id 时，实际归入的项目 ID（便于前端选中同一项目查看树）
	ResolvedDefaultProjectID uint `json:"resolved_default_project_id"`
}

// MigrateFromPoliciesOptions 迁移参数：DefaultProjectID 非 0 时，作为「未写 project_id」策略的目标项目；为 0 则取数据库中首个启用项目。
type MigrateFromPoliciesOptions struct {
	DisableOld         bool
	DefaultProjectID   uint
}

func (s *AlertSubscriptionService) MigrateFromPolicies(ctx context.Context, opts MigrateFromPoliciesOptions) (*SubscriptionMigrationReport, error) {
	disableOld := opts.DisableOld
	rep := &SubscriptionMigrationReport{}
	if s == nil || s.db == nil {
		return rep, nil
	}

	fallbackProject := opts.DefaultProjectID
	if fallbackProject == 0 {
		fp, err := s.firstEnabledProjectID(ctx)
		if err != nil {
			return nil, fmt.Errorf("无法为未指定 project_id 的策略确定目标项目（请先创建业务项目）: %w", err)
		}
		fallbackProject = fp
	}
	rep.ResolvedDefaultProjectID = fallbackProject
	// 旧策略代码已剔除：迁移仅通过表读取历史数据（alert_policies）
	type legacyPolicy struct {
		ID            uint      `gorm:"column:id"`
		Name          string    `gorm:"column:name"`
		Description   string    `gorm:"column:description"`
		Enabled       bool      `gorm:"column:enabled"`
		MatchLabelsJSON string  `gorm:"column:match_labels_json"`
		MatchRegexJSON  string  `gorm:"column:match_regex_json"`
		ChannelsJSON    string  `gorm:"column:channels_json"`
		NotifyResolved  bool    `gorm:"column:notify_resolved"`
		SilenceSeconds  int     `gorm:"column:silence_seconds"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}
	var policies []legacyPolicy
	if err := s.db.WithContext(ctx).Table("alert_policies").Find(&policies).Error; err != nil {
		return nil, err
	}
	rep.PoliciesTotal = len(policies)

	rootByProject := map[uint]*model.AlertSubscriptionNode{}
	getOrCreateRoot := func(projectID uint) (*model.AlertSubscriptionNode, error) {
		if n, ok := rootByProject[projectID]; ok && n != nil {
			return n, nil
		}
		var root model.AlertSubscriptionNode
		err := s.db.WithContext(ctx).
			Where("project_id = ? AND parent_id IS NULL", projectID).
			Order("id ASC").
			First(&root).Error
		if err == nil {
			rootByProject[projectID] = &root
			return &root, nil
		}
		root = model.AlertSubscriptionNode{
			ProjectID:            projectID,
			ParentID:             nil,
			Level:                0,
			Path:                 fmt.Sprintf("/%d", projectID),
			Name:                 "root",
			Code:                 "root",
			MatchLabelsJSON:      "{}",
			MatchRegexJSON:       "{}",
			MatchSeverity:        "",
			Continue:             true,
			Enabled:              true,
			ReceiverGroupIDsJSON: "[]",
			SilenceSeconds:       0,
			NotifyResolved:       true,
		}
		if err2 := s.db.WithContext(ctx).Create(&root).Error; err2 != nil {
			return nil, err2
		}
		rootByProject[projectID] = &root
		return &root, nil
	}

	for i := range policies {
		p := policies[i]
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		projectID := extractProjectIDFromPolicyMatchLabels(p.MatchLabelsJSON)
		if projectID == 0 {
			projectID = fallbackProject
		}
		policyCode := fmt.Sprintf("policy_%d", p.ID)
		var already model.AlertSubscriptionNode
		if err := s.db.WithContext(ctx).
			Where("project_id = ? AND code = ?", projectID, policyCode).
			First(&already).Error; err == nil {
			// 已迁移过（重复执行迁移时避免再建接收组/节点）
			continue
		}

		root, err := getOrCreateRoot(projectID)
		if err != nil {
			return nil, err
		}

		rg := &model.AlertReceiverGroup{
			ProjectID:           projectID,
			Name:                "migrated:" + strings.TrimSpace(p.Name),
			Description:         strings.TrimSpace(p.Description),
			ChannelIDsJSON:      strings.TrimSpace(p.ChannelsJSON),
			EmailRecipientsJSON: "[]",
			EscalationLevel:     0,
			Enabled:             p.Enabled,
		}
		if strings.TrimSpace(rg.ChannelIDsJSON) == "" {
			rg.ChannelIDsJSON = "[]"
		}
		if err := s.db.WithContext(ctx).Create(rg).Error; err != nil {
			return nil, err
		}
		rep.ReceiverGroupsCreated++

		node := &model.AlertSubscriptionNode{
			ProjectID:            projectID,
			ParentID:             &root.ID,
			Level:                root.Level + 1,
			Path:                 fmt.Sprintf("%s/%d", root.Path, root.ID),
			Name:                 strings.TrimSpace(p.Name),
			Code:                 fmt.Sprintf("policy_%d", p.ID),
			MatchLabelsJSON:      safeJSONObj(p.MatchLabelsJSON),
			MatchRegexJSON:       safeJSONObj(p.MatchRegexJSON),
			MatchSeverity:        "",
			Continue:             false,
			Enabled:              p.Enabled,
			ReceiverGroupIDsJSON: fmt.Sprintf("[%d]", rg.ID),
			SilenceSeconds:       p.SilenceSeconds,
			NotifyResolved:       p.NotifyResolved,
		}
		if err := s.db.WithContext(ctx).Create(node).Error; err != nil {
			return nil, err
		}
		rep.NodesCreated++
		rep.PoliciesMigrated++

		if disableOld && p.Enabled {
			_ = s.db.WithContext(ctx).Table("alert_policies").Where("id = ?", p.ID).Update("enabled", false).Error
			rep.PoliciesDisabled++
		}
	}

	s.InvalidateCache()
	return rep, nil
}

func safeJSONObj(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	if raw == "{}" {
		return raw
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "{}"
	}
	bs, _ := json.Marshal(obj)
	return string(bs)
}

func extractProjectIDFromPolicyMatchLabels(raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return 0
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return 0
	}
	v, ok := m["project_id"]
	if !ok {
		return 0
	}
	switch vv := v.(type) {
	case float64:
		if vv > 0 {
			return uint(vv)
		}
	case string:
		n, ok2 := parseLabelUint(vv)
		if ok2 {
			return n
		}
	}
	return 0
}

func (s *AlertSubscriptionService) firstEnabledProjectID(ctx context.Context) (uint, error) {
	var p model.Project
	err := s.db.WithContext(ctx).Where("status = ?", 1).Order("id ASC").First(&p).Error
	if err != nil {
		return 0, err
	}
	if p.ID == 0 {
		return 0, fmt.Errorf("no project")
	}
	return p.ID, nil
}

