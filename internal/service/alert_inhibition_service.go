package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AlertInhibitionService 告警抑制服务
type AlertInhibitionService struct {
	db    *gorm.DB
	redis *redis.Client

	cacheMu          sync.RWMutex
	cachedRules      []model.AlertInhibitionRule
	compiledMatchers map[uint]*inhibitionCompiledMatchers
	cacheVersion     int64
	lastCacheUpdate  time.Time
}

// inhibitionCompiledMatchers 预编译的匹配器
type inhibitionCompiledMatchers struct {
	sourceMatchLabels map[string]string
	sourceMatchRegex  map[string]*regexp.Regexp
	targetMatchLabels map[string]string
	targetMatchRegex  map[string]*regexp.Regexp
	equalLabels       []string
}

// NewAlertInhibitionService 创建告警抑制服务
func NewAlertInhibitionService(db *gorm.DB, redisClient *redis.Client) *AlertInhibitionService {
	svc := &AlertInhibitionService{
		db:               db,
		redis:            redisClient,
		compiledMatchers: make(map[uint]*inhibitionCompiledMatchers),
	}
	// 启动时预热缓存
	_ = svc.refreshCache(context.Background())
	return svc
}

// AlertInhibitionRuleUpsertRequest 创建/更新请求
type AlertInhibitionRuleUpsertRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
	Priority    int    `json:"priority"`

	SourceMatchLabelsJSON string `json:"source_match_labels_json"`
	SourceMatchRegexJSON  string `json:"source_match_regex_json"`
	TargetMatchLabelsJSON string `json:"target_match_labels_json"`
	TargetMatchRegexJSON  string `json:"target_match_regex_json"`
	EqualLabelsJSON       string `json:"equal_labels_json"`

	DurationSeconds int `json:"duration_seconds"`
}

// AlertInhibitionRuleListQuery 列表查询
type AlertInhibitionRuleListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Enabled  *bool  `form:"enabled"`
}

// List 查询抑制规则列表
func (s *AlertInhibitionService) List(ctx context.Context, q AlertInhibitionRuleListQuery) ([]model.AlertInhibitionRule, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertInhibitionRule{})

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

	var list []model.AlertInhibitionRule
	if err := tx.Order("priority ASC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	// 填充解析后的数据
	for i := range list {
		hydrateInhibitionRule(&list[i])
	}

	return list, total, page, pageSize, nil
}

// Create 创建抑制规则
func (s *AlertInhibitionService) Create(ctx context.Context, req AlertInhibitionRuleUpsertRequest) (*model.AlertInhibitionRule, error) {
	if err := validateInhibitionRule(req); err != nil {
		return nil, err
	}

	rule := &model.AlertInhibitionRule{
		Name:                  strings.TrimSpace(req.Name),
		Description:           strings.TrimSpace(req.Description),
		SourceMatchLabelsJSON: strings.TrimSpace(req.SourceMatchLabelsJSON),
		SourceMatchRegexJSON:  strings.TrimSpace(req.SourceMatchRegexJSON),
		TargetMatchLabelsJSON: strings.TrimSpace(req.TargetMatchLabelsJSON),
		TargetMatchRegexJSON:  strings.TrimSpace(req.TargetMatchRegexJSON),
		EqualLabelsJSON:      strings.TrimSpace(req.EqualLabelsJSON),
		DurationSeconds:       req.DurationSeconds,
	}
	if req.Priority <= 0 {
		rule.Priority = 100
	} else {
		rule.Priority = req.Priority
	}
	if req.Enabled == nil {
		rule.Enabled = true
	} else {
		rule.Enabled = *req.Enabled
	}
	if rule.DurationSeconds <= 0 {
		rule.DurationSeconds = 3600 // 默认1小时
	}

	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return nil, err
	}

	s.InvalidateCache()
	hydrateInhibitionRule(rule)
	return rule, nil
}

// Update 更新抑制规则
func (s *AlertInhibitionService) Update(ctx context.Context, id uint, req AlertInhibitionRuleUpsertRequest) (*model.AlertInhibitionRule, error) {
	var rule model.AlertInhibitionRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}

	if err := validateInhibitionRule(req); err != nil {
		return nil, err
	}

	rule.Name = strings.TrimSpace(req.Name)
	rule.Description = strings.TrimSpace(req.Description)
	rule.SourceMatchLabelsJSON = strings.TrimSpace(req.SourceMatchLabelsJSON)
	rule.SourceMatchRegexJSON = strings.TrimSpace(req.SourceMatchRegexJSON)
	rule.TargetMatchLabelsJSON = strings.TrimSpace(req.TargetMatchLabelsJSON)
	rule.TargetMatchRegexJSON = strings.TrimSpace(req.TargetMatchRegexJSON)
	rule.EqualLabelsJSON = strings.TrimSpace(req.EqualLabelsJSON)
	rule.DurationSeconds = req.DurationSeconds
	if req.Priority > 0 {
		rule.Priority = req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if rule.DurationSeconds <= 0 {
		rule.DurationSeconds = 3600
	}

	if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
		return nil, err
	}

	s.InvalidateCache()
	hydrateInhibitionRule(&rule)
	return &rule, nil
}

// Delete 删除抑制规则
func (s *AlertInhibitionService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertInhibitionRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("抑制规则不存在")
	}
	s.InvalidateCache()
	return nil
}

// InvalidateCache 使缓存失效
func (s *AlertInhibitionService) InvalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cachedRules = nil
	s.compiledMatchers = make(map[uint]*inhibitionCompiledMatchers)
	s.cacheVersion++
	s.lastCacheUpdate = time.Time{}
}

// RefreshCache 刷新缓存（公开方法）
func (s *AlertInhibitionService) RefreshCache(ctx context.Context) error {
	return s.refreshCache(ctx)
}

// refreshCache 刷新缓存
func (s *AlertInhibitionService) refreshCache(ctx context.Context) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// 每30秒刷新一次
	if time.Since(s.lastCacheUpdate) < 30*time.Second && len(s.cachedRules) > 0 {
		return nil
	}

	var rules []model.AlertInhibitionRule
	if err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("priority ASC, id ASC").
		Find(&rules).Error; err != nil {
		return err
	}

	s.cachedRules = rules
	s.compiledMatchers = make(map[uint]*inhibitionCompiledMatchers, len(rules))
	for i := range rules {
		hydrateInhibitionRule(&rules[i])
		s.compiledMatchers[rules[i].ID] = compileInhibitionMatchers(&rules[i])
	}
	s.cacheVersion++
	s.lastCacheUpdate = time.Now()
	return nil
}

// ListEnabledRules 获取启用的规则（带缓存）
func (s *AlertInhibitionService) ListEnabledRules(ctx context.Context) ([]model.AlertInhibitionRule, error) {
	if err := s.refreshCache(ctx); err != nil {
		return nil, err
	}
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	result := make([]model.AlertInhibitionRule, len(s.cachedRules))
	copy(result, s.cachedRules)
	return result, nil
}

// CheckSourceMatch 检查告警是否匹配源告警条件
func (s *AlertInhibitionService) CheckSourceMatch(ctx context.Context, labels map[string]string) ([]uint, error) {
	rules, err := s.ListEnabledRules(ctx)
	if err != nil {
		return nil, err
	}

	matched := make([]uint, 0)
	s.cacheMu.RLock()
	matchers := s.compiledMatchers
	s.cacheMu.RUnlock()

	for _, rule := range rules {
		m := matchers[rule.ID]
		if m == nil {
			continue
		}
		if matchLabels(labels, m.sourceMatchLabels, m.sourceMatchRegex) {
			matched = append(matched, rule.ID)
		}
	}

	return matched, nil
}

// CheckInhibition 检查告警是否被抑制
func (s *AlertInhibitionService) CheckInhibition(ctx context.Context, targetLabels map[string]string) (bool, *model.AlertInhibitionEvent, error) {
	if s.redis == nil {
		return false, nil, nil
	}

	rules, err := s.ListEnabledRules(ctx)
	if err != nil {
		return false, nil, err
	}

	s.cacheMu.RLock()
	matchers := s.compiledMatchers
	s.cacheMu.RUnlock()

	now := time.Now()

	for _, rule := range rules {
		m := matchers[rule.ID]
		if m == nil {
			continue
		}

		// 首先检查目标告警是否匹配目标条件
		if !matchLabels(targetLabels, m.targetMatchLabels, m.targetMatchRegex) {
			continue
		}

		// 查询该规则下活跃的源告警
		sourceFingerprints, err := s.getActiveSources(ctx, rule.ID)
		if err != nil {
			continue
		}

		for _, sourceFP := range sourceFingerprints {
			// 获取源告警的标签
			sourceLabels, err := s.getSourceLabels(ctx, rule.ID, sourceFP)
			if err != nil {
				continue
			}

			// 检查equal标签是否匹配
			if !checkEqualLabels(targetLabels, sourceLabels, m.equalLabels) {
				continue
			}

			// 确认源告警是否仍然活跃
			ttl, err := s.redis.TTL(ctx, inhibitionSourceKey(rule.ID, sourceFP)).Result()
			if err != nil || ttl <= 0 {
				continue
			}

			// 被抑制了！
			event := &model.AlertInhibitionEvent{
				RuleID:            rule.ID,
				RuleName:          rule.Name,
				SourceFingerprint: sourceFP,
				TargetFingerprint: extractFingerprint(targetLabels),
				SourceAlertName:   sourceLabels["alertname"],
				TargetAlertName: targetLabels["alertname"],
				StartedAt:         now.Add(-time.Duration(ttl)), // 估算
				EndedAt:           now.Add(ttl),
			}
			return true, event, nil
		}
	}

	return false, nil, nil
}

// RecordSourceAlert 记录源告警触发
func (s *AlertInhibitionService) RecordSourceAlert(ctx context.Context, ruleID uint, fingerprint string, labels map[string]string) error {
	if s.redis == nil {
		return nil
	}

	rule, err := s.getRuleByIDFromCache(ruleID)
	if err != nil {
		return err
	}

	key := inhibitionSourceKey(ruleID, fingerprint)
	data, _ := json.Marshal(labels)

	ttl := time.Duration(rule.DurationSeconds) * time.Second
	return s.redis.Set(ctx, key, string(data), ttl).Err()
}

// ClearSourceAlert 清除源告警
func (s *AlertInhibitionService) ClearSourceAlert(ctx context.Context, ruleID uint, fingerprint string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, inhibitionSourceKey(ruleID, fingerprint)).Err()
}

// getActiveSources 获取规则下所有活跃的源告警指纹
func (s *AlertInhibitionService) getActiveSources(ctx context.Context, ruleID uint) ([]string, error) {
	pattern := inhibitionSourcePattern(ruleID)
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	fingerprints := make([]string, 0, len(keys))
	for _, key := range keys {
		// key格式: alert:inhibition:source:{ruleID}:{fingerprint}
		parts := strings.Split(key, ":")
		if len(parts) >= 5 {
			fingerprints = append(fingerprints, parts[4])
		}
	}
	return fingerprints, nil
}

// getSourceLabels 获取源告警的标签
func (s *AlertInhibitionService) getSourceLabels(ctx context.Context, ruleID uint, fingerprint string) (map[string]string, error) {
	key := inhibitionSourceKey(ruleID, fingerprint)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(data), &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

func (s *AlertInhibitionService) getRuleByIDFromCache(ruleID uint) (*model.AlertInhibitionRule, error) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	for i := range s.cachedRules {
		if s.cachedRules[i].ID == ruleID {
			return &s.cachedRules[i], nil
		}
	}
	return nil, fmt.Errorf("rule not found: %d", ruleID)
}

// 辅助函数

func hydrateInhibitionRule(rule *model.AlertInhibitionRule) {
	if rule == nil {
		return
	}
	rule.SourceMatchLabels = parseMapJSONSafe(rule.SourceMatchLabelsJSON)
	rule.SourceMatchRegex = parseMapJSONSafe(rule.SourceMatchRegexJSON)
	rule.TargetMatchLabels = parseMapJSONSafe(rule.TargetMatchLabelsJSON)
	rule.TargetMatchRegex = parseMapJSONSafe(rule.TargetMatchRegexJSON)
	rule.EqualLabels = parseStringSliceSafe(rule.EqualLabelsJSON)
}

func compileInhibitionMatchers(rule *model.AlertInhibitionRule) *inhibitionCompiledMatchers {
	m := &inhibitionCompiledMatchers{
		sourceMatchLabels: parseMapJSONSafe(rule.SourceMatchLabelsJSON),
		sourceMatchRegex:  compileRegexMapSafe(rule.SourceMatchRegexJSON),
		targetMatchLabels: parseMapJSONSafe(rule.TargetMatchLabelsJSON),
		targetMatchRegex:  compileRegexMapSafe(rule.TargetMatchRegexJSON),
		equalLabels:       parseStringSliceSafe(rule.EqualLabelsJSON),
	}
	return m
}

func matchLabels(labels map[string]string, exact map[string]string, regex map[string]*regexp.Regexp) bool {
	// 精确匹配
	for k, v := range exact {
		if strings.TrimSpace(labels[k]) != strings.TrimSpace(v) {
			return false
		}
	}
	// 正则匹配
	for k, re := range regex {
		if re == nil {
			continue
		}
		if !re.MatchString(strings.TrimSpace(labels[k])) {
			return false
		}
	}
	return true
}

func checkEqualLabels(target, source map[string]string, equalLabels []string) bool {
	for _, key := range equalLabels {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		if strings.TrimSpace(target[k]) != strings.TrimSpace(source[k]) {
			return false
		}
	}
	return true
}

func extractFingerprint(labels map[string]string) string {
	fp := labels["fingerprint"]
	if fp == "" {
		// 根据labels计算简易指纹
		var sb strings.Builder
		for k, v := range labels {
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			sb.WriteString(";")
		}
		return fmt.Sprintf("%x", sb.String())[:32]
	}
	return fp
}

func inhibitionSourceKey(ruleID uint, fingerprint string) string {
	return fmt.Sprintf("alert:inhibition:source:%d:%s", ruleID, fingerprint)
}

func inhibitionSourcePattern(ruleID uint) string {
	return fmt.Sprintf("alert:inhibition:source:%d:*", ruleID)
}

func parseStringSliceSafe(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func validateInhibitionRule(req AlertInhibitionRuleUpsertRequest) error {
	// 源告警和目标告警至少有一个匹配条件
	hasSource := strings.TrimSpace(req.SourceMatchLabelsJSON) != "" &&
		req.SourceMatchLabelsJSON != "{}" ||
		strings.TrimSpace(req.SourceMatchRegexJSON) != "" &&
		req.SourceMatchRegexJSON != "{}"
	hasTarget := strings.TrimSpace(req.TargetMatchLabelsJSON) != "" &&
		req.TargetMatchLabelsJSON != "{}" ||
		strings.TrimSpace(req.TargetMatchRegexJSON) != "" &&
		req.TargetMatchRegexJSON != "{}"

	if !hasSource {
		return apperror.BadRequest("源告警匹配条件不能为空")
	}
	if !hasTarget {
		return apperror.BadRequest("目标告警匹配条件不能为空")
	}

	// 验证equal_labels是有效的JSON数组
	if raw := strings.TrimSpace(req.EqualLabelsJSON); raw != "" && raw != "[]" {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return apperror.BadRequest("equal_labels_json 必须是字符串数组JSON")
		}
	}

	// 验证正则表达式合法
	if m := compileRegexMapSafe(req.SourceMatchRegexJSON); len(m) == 0 &&
		strings.TrimSpace(req.SourceMatchRegexJSON) != "" &&
		req.SourceMatchRegexJSON != "{}" {
		return apperror.BadRequest("源告警正则表达式不合法")
	}
	if m := compileRegexMapSafe(req.TargetMatchRegexJSON); len(m) == 0 &&
		strings.TrimSpace(req.TargetMatchRegexJSON) != "" &&
		req.TargetMatchRegexJSON != "{}" {
		return apperror.BadRequest("目标告警正则表达式不合法")
	}

	return nil
}

// RecordInhibitionEvent 记录抑制事件到数据库（可选，用于审计）
func (s *AlertInhibitionService) RecordInhibitionEvent(ctx context.Context, event *model.AlertInhibitionEvent) error {
	return s.db.WithContext(ctx).Create(event).Error
}
