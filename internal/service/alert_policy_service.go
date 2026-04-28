package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	amconfig "github.com/prometheus/alertmanager/config"
	"gorm.io/gorm"
)

type AlertPolicyListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Enabled  *bool  `form:"enabled"`
}

type AlertPolicyUpsertRequest struct {
	Name            string `json:"name" binding:"required,max=128"`
	Description     string `json:"description"`
	Enabled         *bool  `json:"enabled"`
	Priority        int    `json:"priority"`
	MatchLabelsJSON string `json:"match_labels_json"`
	MatchRegexJSON  string `json:"match_regex_json"`
	ChannelsJSON    string `json:"channels_json"`
	TemplateID      *uint  `json:"template_id"`
	NotifyResolved  *bool  `json:"notify_resolved"`
	SilenceSeconds  int    `json:"silence_seconds"`
}

type AlertPolicyService struct {
	db *gorm.DB
}

func NewAlertPolicyService(db *gorm.DB) *AlertPolicyService {
	return &AlertPolicyService{db: db}
}

func hydrateAlertPolicy(it *model.AlertPolicy) {
	if it == nil {
		return
	}
	it.MatchLabels = parseMapJSON(it.MatchLabelsJSON)
	it.MatchRegex = parseMapJSON(it.MatchRegexJSON)
	it.ChannelIDs = parseUintSliceJSON(it.ChannelsJSON)
}

func (s *AlertPolicyService) List(ctx context.Context, q AlertPolicyListQuery) (list []model.AlertPolicy, total int64, page int, pageSize int, err error) {
	page, pageSize = pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertPolicy{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if q.Enabled != nil {
		tx = tx.Where("enabled = ?", *q.Enabled)
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	if err = tx.Order("priority ASC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	for i := range list {
		hydrateAlertPolicy(&list[i])
	}
	return list, total, page, pageSize, nil
}

func validateLabelMapJSON(raw string, key string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return apperror.BadRequest(fmt.Sprintf("%s 必须是对象JSON: %v", key, err))
	}
	return nil
}

func validateAlertmanagerRouteBySDK(matchLabelsJSON, matchRegexJSON string) error {
	labels := parseMapJSON(matchLabelsJSON)
	regex := parseMapJSON(matchRegexJSON)
	matchers := make([]string, 0, len(labels)+len(regex))
	for k, v := range labels {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		matchers = append(matchers, fmt.Sprintf(`%s="%s"`, k, strings.ReplaceAll(v, `"`, `\"`)))
	}
	for k, v := range regex {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		matchers = append(matchers, fmt.Sprintf(`%s=~"%s"`, k, strings.ReplaceAll(v, `"`, `\"`)))
	}
	cfg := "route:\n  receiver: default\nreceivers:\n  - name: default\n"
	if len(matchers) > 0 {
		cfg = "route:\n  receiver: default\n  routes:\n    - receiver: default\n      matchers:\n"
		for _, m := range matchers {
			cfg += fmt.Sprintf("        - '%s'\n", m)
		}
		cfg += "receivers:\n  - name: default\n"
	}
	if _, err := amconfig.Load(cfg); err != nil {
		return apperror.BadRequest(fmt.Sprintf("策略匹配规则不兼容 Alertmanager: %v", err))
	}
	return nil
}

func validateChannelsJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return apperror.BadRequest(fmt.Sprintf("channels_json 必须是uint数组JSON: %v", err))
	}
	return nil
}

func (s *AlertPolicyService) Create(ctx context.Context, req AlertPolicyUpsertRequest) (*model.AlertPolicy, error) {
	if err := validateLabelMapJSON(req.MatchLabelsJSON, "match_labels_json"); err != nil {
		return nil, err
	}
	if err := validateLabelMapJSON(req.MatchRegexJSON, "match_regex_json"); err != nil {
		return nil, err
	}
	if err := validateChannelsJSON(req.ChannelsJSON); err != nil {
		return nil, err
	}
	if err := validateAlertmanagerRouteBySDK(req.MatchLabelsJSON, req.MatchRegexJSON); err != nil {
		return nil, err
	}
	it := &model.AlertPolicy{
		Name:            strings.TrimSpace(req.Name),
		Description:     strings.TrimSpace(req.Description),
		MatchLabelsJSON: strings.TrimSpace(req.MatchLabelsJSON),
		MatchRegexJSON:  strings.TrimSpace(req.MatchRegexJSON),
		ChannelsJSON:    strings.TrimSpace(req.ChannelsJSON),
		TemplateID:      req.TemplateID,
		SilenceSeconds:  req.SilenceSeconds,
	}
	if req.Priority <= 0 {
		it.Priority = 100
	} else {
		it.Priority = req.Priority
	}
	if req.Enabled == nil {
		it.Enabled = true
	} else {
		it.Enabled = *req.Enabled
	}
	if req.NotifyResolved == nil {
		it.NotifyResolved = true
	} else {
		it.NotifyResolved = *req.NotifyResolved
	}
	if err := s.db.WithContext(ctx).Create(it).Error; err != nil {
		return nil, err
	}
	hydrateAlertPolicy(it)
	return it, nil
}

func (s *AlertPolicyService) Update(ctx context.Context, id uint, req AlertPolicyUpsertRequest) (*model.AlertPolicy, error) {
	var it model.AlertPolicy
	if err := s.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	if err := validateLabelMapJSON(req.MatchLabelsJSON, "match_labels_json"); err != nil {
		return nil, err
	}
	if err := validateLabelMapJSON(req.MatchRegexJSON, "match_regex_json"); err != nil {
		return nil, err
	}
	if err := validateChannelsJSON(req.ChannelsJSON); err != nil {
		return nil, err
	}
	if err := validateAlertmanagerRouteBySDK(req.MatchLabelsJSON, req.MatchRegexJSON); err != nil {
		return nil, err
	}
	it.Name = strings.TrimSpace(req.Name)
	it.Description = strings.TrimSpace(req.Description)
	it.MatchLabelsJSON = strings.TrimSpace(req.MatchLabelsJSON)
	it.MatchRegexJSON = strings.TrimSpace(req.MatchRegexJSON)
	it.ChannelsJSON = strings.TrimSpace(req.ChannelsJSON)
	it.TemplateID = req.TemplateID
	it.SilenceSeconds = req.SilenceSeconds
	if req.Priority <= 0 {
		it.Priority = 100
	} else {
		it.Priority = req.Priority
	}
	if req.Enabled != nil {
		it.Enabled = *req.Enabled
	}
	if req.NotifyResolved != nil {
		it.NotifyResolved = *req.NotifyResolved
	}
	if err := s.db.WithContext(ctx).Save(&it).Error; err != nil {
		return nil, err
	}
	hydrateAlertPolicy(&it)
	return &it, nil
}

func (s *AlertPolicyService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.AlertPolicy{}, id).Error
}

func parseMapJSON(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	_ = json.Unmarshal([]byte(raw), &out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseUintSliceJSON(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []uint
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func (s *AlertPolicyService) ListEnabled(ctx context.Context) ([]model.AlertPolicy, error) {
	var list []model.AlertPolicy
	if err := s.db.WithContext(ctx).
		Model(&model.AlertPolicy{}).
		Where("enabled = ?", true).
		Order("priority ASC, id ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		hydrateAlertPolicy(&list[i])
	}
	return list, nil
}

func (s *AlertPolicyService) MatchPolicyChannels(policy model.AlertPolicy, labels map[string]string) []uint {
	matchLabels := parseMapJSON(policy.MatchLabelsJSON)
	for k, v := range matchLabels {
		if strings.TrimSpace(labels[k]) != strings.TrimSpace(v) {
			return nil
		}
	}
	matchRegex := parseMapJSON(policy.MatchRegexJSON)
	for k, v := range matchRegex {
		re, err := regexp.Compile(strings.TrimSpace(v))
		if err != nil {
			return nil
		}
		if !re.MatchString(strings.TrimSpace(labels[k])) {
			return nil
		}
	}
	return parseUintSliceJSON(policy.ChannelsJSON)
}

func (s *AlertPolicyService) MatchPolicy(policy model.AlertPolicy, labels map[string]string) bool {
	return len(s.MatchPolicyChannels(policy, labels)) > 0
}
