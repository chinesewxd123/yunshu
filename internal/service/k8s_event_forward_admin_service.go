package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
)

type K8sEventForwardAdminService struct {
	db *gorm.DB
}

func NewK8sEventForwardAdminService(db *gorm.DB) *K8sEventForwardAdminService {
	return &K8sEventForwardAdminService{db: db}
}

type K8sEventForwardRuleListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// K8sEventForwardRuleUpsertRequest 规则创建/更新请求（避免 Save 覆盖 created_at 为零值）。
type K8sEventForwardRuleUpsertRequest struct {
	Name           string `json:"name" binding:"required,max=100"`
	Description    string `json:"description"`
	ClusterIDs     string `json:"cluster_ids" binding:"required"`
	WebhookURL     string `json:"webhook_url"`
	Enabled        *bool  `json:"enabled"`
	RuleNamespaces string `json:"rule_namespaces"`
	RuleNames      string `json:"rule_names"`
	RuleReasons    string `json:"rule_reasons"`
	RuleReverse    *bool  `json:"rule_reverse"`
}

func (s *K8sEventForwardAdminService) ListRules(ctx context.Context, q K8sEventForwardRuleListQuery) (*pagination.Result[model.K8sEventForwardRule], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	var total int64
	if err := s.db.WithContext(ctx).Model(&model.K8sEventForwardRule{}).Count(&total).Error; err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "ListRules", err)
	}
	var list []model.K8sEventForwardRule
	err := s.db.WithContext(ctx).Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "ListRules", err)
	}
	return &pagination.Result[model.K8sEventForwardRule]{
		List: list, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (s *K8sEventForwardAdminService) GetRule(ctx context.Context, id uint) (*model.K8sEventForwardRule, error) {
	var rule model.K8sEventForwardRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, svcerr.Pass("k8s.event-forward", "GetRule", err)
	}
	return &rule, nil
}

func (s *K8sEventForwardAdminService) CreateRule(ctx context.Context, req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	rule, err := normalizeK8sEventForwardRule(req)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "CreateRule", err)
	}
	return rule, nil
}

func (s *K8sEventForwardAdminService) UpdateRule(ctx context.Context, id uint, req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	var existing model.K8sEventForwardRule
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, svcerr.Pass("k8s.event-forward", "UpdateRule", err)
	}
	rule, err := normalizeK8sEventForwardRule(req)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	// Select 显式字段，确保 enabled=false 也会更新，且不触碰 created_at
	if err := s.db.WithContext(ctx).Model(&existing).
		Select("Name", "Description", "ClusterIDs", "WebhookURL", "Enabled",
			"RuleNamespaces", "RuleNames", "RuleReasons", "RuleReverse", "UpdatedAt").
		Updates(rule).Error; err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "UpdateRule", err)
	}
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "UpdateRule", err)
	}
	return &existing, nil
}

func (s *K8sEventForwardAdminService) DeleteRule(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.K8sEventForwardRule{}, id)
	if res.Error != nil {
		return svcerr.Pass("k8s.event-forward", "DeleteRule", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *K8sEventForwardAdminService) GetSettings(ctx context.Context) (*model.K8sEventForwardSetting, error) {
	var st model.K8sEventForwardSetting
	err := s.db.WithContext(ctx).First(&st, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		st.ID = 1
		return &st, nil
	}
	if err != nil {
		return nil, svcerr.Pass("k8s.event-forward", "GetSettings", err)
	}
	return &st, nil
}

func (s *K8sEventForwardAdminService) UpdateSettings(ctx context.Context, st *model.K8sEventForwardSetting) error {
	if st == nil {
		return constants.ErrBadRequestWithMsg("参数无效")
	}
	st.ID = 1
	existing, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}
	if existing.ID == 0 {
		return s.db.WithContext(ctx).Create(st).Error
	}
	return s.db.WithContext(ctx).Model(&model.K8sEventForwardSetting{}).Where("id = ?", 1).
		Select("ProcessIntervalSeconds", "BatchSize", "MaxRetries", "WatcherBufferSize", "UpdatedAt").
		Updates(st).Error
}

func normalizeK8sEventForwardRule(req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("规则名称不能为空")
	}
	clusterIDs := strings.TrimSpace(req.ClusterIDs)
	if clusterIDs == "" {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一个目标集群")
	}
	ns, err := normalizeJSONArrayField(req.RuleNamespaces, "rule_namespaces")
	if err != nil {
		return nil, err
	}
	names, err := normalizeJSONArrayField(req.RuleNames, "rule_names")
	if err != nil {
		return nil, err
	}
	reasons, err := normalizeJSONArrayField(req.RuleReasons, "rule_reasons")
	if err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	reverse := false
	if req.RuleReverse != nil {
		reverse = *req.RuleReverse
	}
	return &model.K8sEventForwardRule{
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		ClusterIDs:     clusterIDs,
		WebhookURL:     strings.TrimSpace(req.WebhookURL),
		Enabled:        enabled,
		RuleNamespaces: ns,
		RuleNames:      names,
		RuleReasons:    reasons,
		RuleReverse:    reverse,
	}, nil
}

func normalizeJSONArrayField(raw, field string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		s = "[]"
	}
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return "", constants.ErrBadRequestWithMsg(field + " 须为 JSON 字符串数组，例如 [\"kube-system\"]")
	}
	out, err := json.Marshal(arr)
	if err != nil {
		return "", constants.ErrBadRequestWithMsg(field + " JSON 无效")
	}
	return string(out), nil
}
