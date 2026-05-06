package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

// AlertSubscriptionService 订阅树服务
type AlertSubscriptionService struct {
	db           *gorm.DB
	cacheMu      sync.RWMutex
	rootNodes    map[uint][]*CachedSubscriptionNode // projectID -> root nodes
	allNodes     map[uint]*CachedSubscriptionNode   // nodeID -> node
	cacheVersion int64
	lastCache    time.Time
}

// CachedSubscriptionNode 缓存的订阅树节点
type CachedSubscriptionNode struct {
	ID                 uint
	ProjectID          uint
	ParentID           *uint
	Level              int
	Path               string
	Name               string
	Code               string
	Continue           bool
	Enabled            bool
	MatchLabels        map[string]string
	MatchRegex         map[string]*regexp.Regexp
	MatchSeverity      string
	ReceiverGroupIDs   []uint
	SilenceSeconds     int
	NotifyResolved     bool
	Children           []*CachedSubscriptionNode
}

// NewAlertSubscriptionService 创建订阅服务
func NewAlertSubscriptionService(db *gorm.DB) *AlertSubscriptionService {
	svc := &AlertSubscriptionService{
		db:        db,
		rootNodes: make(map[uint][]*CachedSubscriptionNode),
		allNodes:  make(map[uint]*CachedSubscriptionNode),
	}
	// 启动时预热缓存
	_ = svc.refreshCache(context.Background())
	return svc
}

// AlertSubscriptionNodeUpsertRequest 创建/更新请求
type AlertSubscriptionNodeUpsertRequest struct {
	ProjectID       uint               `json:"project_id" binding:"required"`
	ParentID        *uint              `json:"parent_id"`
	Name            string             `json:"name" binding:"required,max=128"`
	Code            string             `json:"code" binding:"omitempty,max=64"`
	MatchLabelsJSON string             `json:"match_labels_json"`
	MatchRegexJSON  string             `json:"match_regex_json"`
	MatchSeverity   string             `json:"match_severity" binding:"omitempty,max=32"`
	Continue        bool               `json:"continue"`
	Enabled         *bool              `json:"enabled"`
	ReceiverGroupIDsJSON string        `json:"receiver_group_ids_json"`
	SilenceSeconds  int                `json:"silence_seconds"`
	NotifyResolved  *bool              `json:"notify_resolved"`
}

// AlertSubscriptionNodeListQuery 列表查询
type AlertSubscriptionNodeListQuery struct {
	ProjectID uint   `form:"project_id"`
	ParentID  *uint  `form:"parent_id"`
	Keyword   string `form:"keyword"`
	Enabled   *bool  `form:"enabled"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

// ListNodes 查询订阅树节点列表
func (s *AlertSubscriptionService) ListNodes(ctx context.Context, q AlertSubscriptionNodeListQuery) ([]model.AlertSubscriptionNode, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{})

	if q.ProjectID > 0 {
		tx = tx.Where("project_id = ?", q.ProjectID)
	}
	if q.ParentID != nil {
		if *q.ParentID == 0 {
			tx = tx.Where("parent_id IS NULL")
		} else {
			tx = tx.Where("parent_id = ?", *q.ParentID)
		}
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if q.Enabled != nil {
		tx = tx.Where("enabled = ?", *q.Enabled)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	var list []model.AlertSubscriptionNode
	if err := tx.Order("level ASC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	for i := range list {
		hydrateSubscriptionNode(&list[i])
	}

	return list, total, page, pageSize, nil
}

// GetNodeByID 获取单个节点（用于节点移动等场景）。
func (s *AlertSubscriptionService) GetNodeByID(ctx context.Context, id uint) (*model.AlertSubscriptionNode, error) {
	var node model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, err
	}
	hydrateSubscriptionNode(&node)
	return &node, nil
}

// GetNodeTree 获取完整订阅树（按项目）
// 管理界面需展示启用与停用节点，故不按 enabled 过滤（路由匹配缓存仍只加载 enabled=true）。
func (s *AlertSubscriptionService) GetNodeTree(ctx context.Context, projectID uint) ([]model.AlertSubscriptionNode, error) {
	var nodes []model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("path ASC, id ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	for i := range nodes {
		hydrateSubscriptionNode(&nodes[i])
	}

	return buildNodeTree(nodes), nil
}

// buildNodeTree 将扁平列表构建为树结构
func buildNodeTree(nodes []model.AlertSubscriptionNode) []model.AlertSubscriptionNode {
	nodeMap := make(map[uint]*model.AlertSubscriptionNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	var roots []model.AlertSubscriptionNode
	for i := range nodes {
		if nodes[i].ParentID == nil {
			roots = append(roots, nodes[i])
		} else {
			if parent, ok := nodeMap[*nodes[i].ParentID]; ok {
				parent.Children = append(parent.Children, nodes[i])
			}
		}
	}
	return roots
}

// CreateNode 创建订阅节点
func (s *AlertSubscriptionService) CreateNode(ctx context.Context, req AlertSubscriptionNodeUpsertRequest) (*model.AlertSubscriptionNode, error) {
	if err := validateSubscriptionNode(req); err != nil {
		return nil, err
	}

	// 计算层级和路径
	level := 0
	var parent *model.AlertSubscriptionNode
	var path string

	if req.ParentID != nil && *req.ParentID > 0 {
		parent = &model.AlertSubscriptionNode{}
		if err := s.db.WithContext(ctx).First(parent, *req.ParentID).Error; err != nil {
			return nil, apperror.BadRequest("父节点不存在")
		}
		if parent.ProjectID != req.ProjectID {
			return nil, apperror.BadRequest("父节点不属于该项目")
		}
		level = parent.Level + 1
		path = fmt.Sprintf("%s/%d", parent.Path, req.ParentID)
	} else {
		path = fmt.Sprintf("/%d", req.ProjectID)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	notifyResolved := true
	if req.NotifyResolved != nil {
		notifyResolved = *req.NotifyResolved
	}

	node := &model.AlertSubscriptionNode{
		ProjectID:            req.ProjectID,
		ParentID:             req.ParentID,
		Level:                level,
		Path:                 path,
		Name:                 strings.TrimSpace(req.Name),
		Code:                 strings.TrimSpace(req.Code),
		MatchLabelsJSON:      strings.TrimSpace(req.MatchLabelsJSON),
		MatchRegexJSON:       strings.TrimSpace(req.MatchRegexJSON),
		MatchSeverity:        strings.TrimSpace(req.MatchSeverity),
		Continue:             req.Continue,
		Enabled:              enabled,
		ReceiverGroupIDsJSON: strings.TrimSpace(req.ReceiverGroupIDsJSON),
		SilenceSeconds:       req.SilenceSeconds,
		NotifyResolved:       notifyResolved,
	}

	if err := s.db.WithContext(ctx).Create(node).Error; err != nil {
		return nil, err
	}

	s.InvalidateCache()
	hydrateSubscriptionNode(node)
	return node, nil
}

// UpdateNode 更新订阅节点
func (s *AlertSubscriptionService) UpdateNode(ctx context.Context, id uint, req AlertSubscriptionNodeUpsertRequest) (*model.AlertSubscriptionNode, error) {
	var node model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, apperror.NotFound("订阅节点不存在")
	}

	// 不允许修改所属项目
	if req.ProjectID > 0 && req.ProjectID != node.ProjectID {
		return nil, apperror.BadRequest("不能修改订阅节点所属项目")
	}

	// 不允许将自己设为自己的父节点，或造成循环依赖
	if req.ParentID != nil && *req.ParentID > 0 {
		if *req.ParentID == id {
			return nil, apperror.BadRequest("不能将节点设为自身的父节点")
		}
		// 检查是否会导致循环
		if s.isDescendant(ctx, id, *req.ParentID) {
			return nil, apperror.BadRequest("不能将节点移动到自己的子树下")
		}
	}

	if err := validateSubscriptionNode(req); err != nil {
		return nil, err
	}

	// 重新计算层级和路径
	if req.ParentID != node.ParentID {
		level := 0
		var path string
		if req.ParentID != nil && *req.ParentID > 0 {
			parent := &model.AlertSubscriptionNode{}
			if err := s.db.WithContext(ctx).First(parent, *req.ParentID).Error; err != nil {
				return nil, apperror.BadRequest("父节点不存在")
			}
			level = parent.Level + 1
			path = fmt.Sprintf("%s/%d", parent.Path, *req.ParentID)
		} else {
			path = fmt.Sprintf("/%d", node.ProjectID)
		}
		node.ParentID = req.ParentID
		node.Level = level
		node.Path = path
		// 需要更新所有子节点的路径
		s.updateDescendantsPath(ctx, id, path)
	}

	node.Name = strings.TrimSpace(req.Name)
	node.Code = strings.TrimSpace(req.Code)
	node.MatchLabelsJSON = strings.TrimSpace(req.MatchLabelsJSON)
	node.MatchRegexJSON = strings.TrimSpace(req.MatchRegexJSON)
	node.MatchSeverity = strings.TrimSpace(req.MatchSeverity)
	node.Continue = req.Continue
	node.ReceiverGroupIDsJSON = strings.TrimSpace(req.ReceiverGroupIDsJSON)
	node.SilenceSeconds = req.SilenceSeconds

	if req.Enabled != nil {
		node.Enabled = *req.Enabled
	}
	if req.NotifyResolved != nil {
		node.NotifyResolved = *req.NotifyResolved
	}

	if err := s.db.WithContext(ctx).Save(&node).Error; err != nil {
		return nil, err
	}

	s.InvalidateCache()
	hydrateSubscriptionNode(&node)
	return &node, nil
}

// DeleteNode 删除订阅节点
func (s *AlertSubscriptionService) DeleteNode(ctx context.Context, id uint) error {
	// 检查是否有子节点
	var childCount int64
	if err := s.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).
		Where("parent_id = ?", id).
		Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return apperror.BadRequest("该节点有子节点，请先删除子节点")
	}

	res := s.db.WithContext(ctx).Delete(&model.AlertSubscriptionNode{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("订阅节点不存在")
	}

	s.InvalidateCache()
	return nil
}

// MatchRoute 匹配告警到订阅树路由
// 返回匹配到的接收组ID列表，以及是否继续匹配的标志
func (s *AlertSubscriptionService) MatchRoute(projectID uint, labels map[string]string, severity string) ([]uint, bool) {
	// 兼容旧调用：默认仅用于 firing
	res, ok := s.MatchRouteDetailed(context.Background(), projectID, labels, severity, "firing")
	return res.ReceiverGroupIDs, ok
}

// nodeMatches 检查节点是否匹配告警
func (s *AlertSubscriptionService) nodeMatches(node *CachedSubscriptionNode, labels map[string]string, severity, status string) bool {
	// resolved 是否通知
	if strings.EqualFold(strings.TrimSpace(status), "resolved") && !node.NotifyResolved {
		return false
	}
	// 严重级别匹配
	if node.MatchSeverity != "" && !strings.EqualFold(node.MatchSeverity, severity) {
		return false
	}

	// 精确标签匹配
	for k, v := range node.MatchLabels {
		if strings.TrimSpace(labels[k]) != strings.TrimSpace(v) {
			return false
		}
	}

	// 正则标签匹配
	for k, re := range node.MatchRegex {
		if re == nil {
			continue
		}
		if !re.MatchString(strings.TrimSpace(labels[k])) {
			return false
		}
	}

	return true
}

// InvalidateCache 使缓存失效
func (s *AlertSubscriptionService) InvalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.rootNodes = make(map[uint][]*CachedSubscriptionNode)
	s.allNodes = make(map[uint]*CachedSubscriptionNode)
	s.cacheVersion++
	s.lastCache = time.Time{}
}

// refreshCache 刷新缓存
func (s *AlertSubscriptionService) refreshCache(ctx context.Context) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// 每30秒刷新一次
	if time.Since(s.lastCache) < 30*time.Second && len(s.allNodes) > 0 {
		return nil
	}

	var nodes []model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("path ASC, level ASC, id ASC").
		Find(&nodes).Error; err != nil {
		return err
	}

	// 构建缓存节点
	cachedMap := make(map[uint]*CachedSubscriptionNode)
	for i := range nodes {
		hydrateSubscriptionNode(&nodes[i])
		cached := &CachedSubscriptionNode{
			ID:               nodes[i].ID,
			ProjectID:        nodes[i].ProjectID,
			ParentID:         nodes[i].ParentID,
			Level:            nodes[i].Level,
			Path:             nodes[i].Path,
			Name:             nodes[i].Name,
			Code:             nodes[i].Code,
			Continue:         nodes[i].Continue,
			Enabled:          nodes[i].Enabled,
			MatchLabels:      nodes[i].MatchLabels,
			MatchSeverity:    nodes[i].MatchSeverity,
			ReceiverGroupIDs: nodes[i].ReceiverGroupIDs,
			SilenceSeconds:   nodes[i].SilenceSeconds,
			NotifyResolved:   nodes[i].NotifyResolved,
			MatchRegex:       make(map[string]*regexp.Regexp),
			Children:         []*CachedSubscriptionNode{},
		}

		// 编译正则
		for k, v := range nodes[i].MatchRegex {
			if re, err := regexp.Compile(v); err == nil {
				cached.MatchRegex[k] = re
			}
		}

		cachedMap[nodes[i].ID] = cached
	}

	// 构建树结构
	roots := make(map[uint][]*CachedSubscriptionNode)
	for _, node := range cachedMap {
		if node.ParentID == nil {
			roots[node.ProjectID] = append(roots[node.ProjectID], node)
		} else {
			if parent, ok := cachedMap[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	// 按优先级排序子节点
	for _, node := range cachedMap {
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].ID < node.Children[j].ID
		})
	}

	s.rootNodes = roots
	s.allNodes = cachedMap
	s.cacheVersion++
	s.lastCache = time.Now()
	return nil
}

// isDescendant 检查target是否是node的后代
func (s *AlertSubscriptionService) isDescendant(ctx context.Context, nodeID, targetID uint) bool {
	var children []model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).Where("parent_id = ?", nodeID).Find(&children).Error; err != nil {
		return false
	}

	for _, child := range children {
		if child.ID == targetID {
			return true
		}
		if s.isDescendant(ctx, child.ID, targetID) {
			return true
		}
	}
	return false
}

// updateDescendantsPath 更新所有后代节点的路径
func (s *AlertSubscriptionService) updateDescendantsPath(ctx context.Context, parentID uint, parentPath string) {
	var children []model.AlertSubscriptionNode
	if err := s.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return
	}

	for _, child := range children {
		newPath := fmt.Sprintf("%s/%d", parentPath, child.ID)
		updates := map[string]interface{}{
			"path":  newPath,
			"level": len(strings.Split(newPath, "/")) - 1,
		}
		_ = s.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).Where("id = ?", child.ID).Updates(updates)
		s.updateDescendantsPath(ctx, child.ID, newPath)
	}
}

// validateSubscriptionNode 验证订阅节点
func validateSubscriptionNode(req AlertSubscriptionNodeUpsertRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return apperror.BadRequest("节点名称不能为空")
	}

	// 验证正则表达式合法
	if raw := strings.TrimSpace(req.MatchRegexJSON); raw != "" && raw != "{}" {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return apperror.BadRequest("match_regex_json 格式错误")
		}
		for k, v := range m {
			if _, err := regexp.Compile(v); err != nil {
				return apperror.BadRequest(fmt.Sprintf("正则表达式错误 [%s]: %v", k, err))
			}
		}
	}

	return nil
}

// hydrateSubscriptionNode 填充订阅节点的非数据库字段
func hydrateSubscriptionNode(node *model.AlertSubscriptionNode) {
	if node == nil {
		return
	}
	node.MatchLabels = parseMapJSON(node.MatchLabelsJSON)
	node.MatchRegex = parseMapJSON(node.MatchRegexJSON)
	node.ReceiverGroupIDs = parseUintSliceJSON(node.ReceiverGroupIDsJSON)
}

// AlertRouteResult 路由结果
type AlertRouteResult struct {
	ReceiverGroupIDs []uint
	MatchedPath      string
	MatchedNodeIDs   []uint
	MatchedNodeNames []string
	SilenceSeconds   int
}

// RefreshIfNeeded 刷新缓存（内部带 30s 去抖）
func (s *AlertSubscriptionService) RefreshIfNeeded(ctx context.Context) error {
	return s.refreshCache(ctx)
}

// MatchRouteDetailed 匹配订阅树并返回路由详情（用于投递与审计）。
func (s *AlertSubscriptionService) MatchRouteDetailed(ctx context.Context, projectID uint, labels map[string]string, severity, status string) (AlertRouteResult, bool) {
	_ = s.RefreshIfNeeded(ctx)
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	roots, ok := s.rootNodes[projectID]
	if !ok || len(roots) == 0 {
		return AlertRouteResult{}, false
	}

	out := AlertRouteResult{}
	var anyMatched bool
	for _, root := range roots {
		res, matched := s.matchNodeRecursiveDetailed(root, labels, severity, status)
		if !matched {
			continue
		}
		anyMatched = true
		out.ReceiverGroupIDs = append(out.ReceiverGroupIDs, res.ReceiverGroupIDs...)
		out.MatchedNodeIDs = append(out.MatchedNodeIDs, res.MatchedNodeIDs...)
		out.MatchedNodeNames = append(out.MatchedNodeNames, res.MatchedNodeNames...)
		if res.SilenceSeconds > out.SilenceSeconds {
			out.SilenceSeconds = res.SilenceSeconds
		}
		if out.MatchedPath == "" && res.MatchedPath != "" {
			out.MatchedPath = res.MatchedPath
		}
	}
	if !anyMatched {
		return AlertRouteResult{}, false
	}
	// 去重
	out.ReceiverGroupIDs = uniqUint(out.ReceiverGroupIDs)
	out.MatchedNodeIDs = uniqUint(out.MatchedNodeIDs)
	out.MatchedNodeNames = uniqStrings(out.MatchedNodeNames)
	return out, len(out.ReceiverGroupIDs) > 0
}

func uniqUint(in []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(in))
	for _, v := range in {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func uniqStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func (s *AlertSubscriptionService) matchNodeRecursiveDetailed(node *CachedSubscriptionNode, labels map[string]string, severity, status string) (AlertRouteResult, bool) {
	if node == nil || !node.Enabled {
		return AlertRouteResult{}, false
	}
	if !s.nodeMatches(node, labels, severity, status) {
		return AlertRouteResult{}, false
	}
	res := AlertRouteResult{
		ReceiverGroupIDs: append([]uint{}, node.ReceiverGroupIDs...),
		MatchedPath:      node.Path,
		MatchedNodeIDs:   []uint{node.ID},
		MatchedNodeNames: []string{node.Name},
		SilenceSeconds:   node.SilenceSeconds,
	}
	if !node.Continue {
		return res, true
	}
	for _, child := range node.Children {
		cr, ok := s.matchNodeRecursiveDetailed(child, labels, severity, status)
		if !ok {
			continue
		}
		res.ReceiverGroupIDs = append(res.ReceiverGroupIDs, cr.ReceiverGroupIDs...)
		res.MatchedNodeIDs = append(res.MatchedNodeIDs, cr.MatchedNodeIDs...)
		res.MatchedNodeNames = append(res.MatchedNodeNames, cr.MatchedNodeNames...)
		if cr.SilenceSeconds > res.SilenceSeconds {
			res.SilenceSeconds = cr.SilenceSeconds
		}
	}
	return res, true
}
