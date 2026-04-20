package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/repository"

	"gorm.io/gorm"
)

type DepartmentService struct {
	repo     *repository.DepartmentRepository
	userRepo *repository.UserRepository
}

func NewDepartmentService(repo *repository.DepartmentRepository, userRepo *repository.UserRepository) *DepartmentService {
	return &DepartmentService{repo: repo, userRepo: userRepo}
}

func (s *DepartmentService) actorDepartmentScopeIDs(ctx context.Context, actor *auth.CurrentUser) ([]uint, error) {
	if actor == nil || actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return nil, nil
	}
	return s.repo.ListDescendantIDsAndSelf(ctx, *actor.DepartmentID)
}

func (s *DepartmentService) hasDepartmentAccess(ctx context.Context, actor *auth.CurrentUser, departmentID uint) (bool, error) {
	if actor == nil {
		return false, nil
	}
	if isSuperAdmin(actor.RoleCodes) {
		return true, nil
	}
	scope, err := s.actorDepartmentScopeIDs(ctx, actor)
	if err != nil {
		return false, err
	}
	if len(scope) == 0 {
		return false, nil
	}
	return slices.Contains(scope, departmentID), nil
}

type DepartmentCreateRequest struct {
	ParentID *uint  `json:"parent_id"`
	Name     string `json:"name" binding:"required,max=128"`
	Code     string `json:"code" binding:"required,max=64"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	LeaderID *uint  `json:"leader_id"`
	Phone    string `json:"phone" binding:"omitempty,max=32"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DepartmentUpdateRequest struct {
	ParentID *uint  `json:"parent_id"`
	Name     string `json:"name" binding:"required,max=128"`
	Code     string `json:"code" binding:"required,max=64"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	LeaderID *uint  `json:"leader_id"`
	Phone    string `json:"phone" binding:"omitempty,max=32"`
	Email    string `json:"email" binding:"omitempty,email,max=128"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DepartmentDetailResponse struct {
	ID         uint                       `json:"id"`
	ParentID   *uint                      `json:"parent_id,omitempty"`
	Name       string                     `json:"name"`
	Code       string                     `json:"code"`
	Ancestors  string                     `json:"ancestors"`
	Level      int                        `json:"level"`
	Sort       int                        `json:"sort"`
	Status     int                        `json:"status"`
	LeaderID   *uint                      `json:"leader_id,omitempty"`
	LeaderName string                     `json:"leader_name"`
	Phone      string                     `json:"phone"`
	Email      string                     `json:"email"`
	Remark     string                     `json:"remark"`
	UserCount  int64                      `json:"user_count"`
	CreatedAt  string                     `json:"created_at"`
	UpdatedAt  string                     `json:"updated_at"`
	Children   []DepartmentDetailResponse `json:"children,omitempty"`
}

func (s *DepartmentService) Tree(ctx context.Context) ([]DepartmentDetailResponse, error) {
	all, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	leaderMap, err := s.buildLeaderMap(ctx, all)
	if err != nil {
		return nil, err
	}
	userCountMap, err := s.buildUserCountMap(ctx)
	if err != nil {
		return nil, err
	}
	return buildDepartmentTree(all, leaderMap, userCountMap), nil
}

func (s *DepartmentService) TreeByActor(ctx context.Context, actor *auth.CurrentUser) ([]DepartmentDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	all, err := s.Tree(ctx)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin(actor.RoleCodes) {
		return all, nil
	}
	if actor.DepartmentID == nil || *actor.DepartmentID == 0 {
		return []DepartmentDetailResponse{}, nil
	}
	var walk func([]DepartmentDetailResponse) *DepartmentDetailResponse
	walk = func(nodes []DepartmentDetailResponse) *DepartmentDetailResponse {
		for _, node := range nodes {
			if node.ID == *actor.DepartmentID {
				cp := node
				return &cp
			}
			if len(node.Children) == 0 {
				continue
			}
			if found := walk(node.Children); found != nil {
				return found
			}
		}
		return nil
	}
	found := walk(all)
	if found == nil {
		return []DepartmentDetailResponse{}, nil
	}
	return []DepartmentDetailResponse{*found}, nil
}

func (s *DepartmentService) Detail(ctx context.Context, id uint) (*DepartmentDetailResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("部门不存在")
		}
		return nil, err
	}
	resp := s.toResponse(ctx, *item)
	return &resp, nil
}

func (s *DepartmentService) DetailByActor(ctx context.Context, actor *auth.CurrentUser, id uint) (*DepartmentDetailResponse, error) {
	ok, err := s.hasDepartmentAccess(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("无访问权限")
	}
	return s.Detail(ctx, id)
}

func (s *DepartmentService) Create(ctx context.Context, req DepartmentCreateRequest) (*DepartmentDetailResponse, error) {
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if name == "" || code == "" {
		return nil, apperror.BadRequest("部门名称和编码不能为空")
	}
	if err := s.ensureLeaderExists(ctx, req.LeaderID); err != nil {
		return nil, err
	}
	if exists, err := s.repo.ExistsByCode(ctx, code, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, apperror.Conflict("部门编码已存在")
	}
	if exists, err := s.repo.ExistsByNameInParent(ctx, req.ParentID, name, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, apperror.Conflict("同级部门名称已存在")
	}

	ancestors, level, err := s.resolveAncestorsAndLevel(ctx, req.ParentID)
	if err != nil {
		return nil, err
	}
	status := req.Status
	if status != 0 {
		status = 1
	}
	item := model.Department{
		ParentID:  req.ParentID,
		Name:      name,
		Code:      code,
		Ancestors: ancestors,
		Level:     level,
		Sort:      req.Sort,
		Status:    status,
		LeaderID:  req.LeaderID,
		Phone:     strings.TrimSpace(req.Phone),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Remark:    strings.TrimSpace(req.Remark),
	}
	if err = s.repo.Create(ctx, &item); err != nil {
		return nil, err
	}
	resp := s.toResponse(ctx, item)
	return &resp, nil
}

func (s *DepartmentService) CreateByActor(ctx context.Context, actor *auth.CurrentUser, req DepartmentCreateRequest) (*DepartmentDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	if isSuperAdmin(actor.RoleCodes) {
		return s.Create(ctx, req)
	}
	if req.ParentID == nil {
		return nil, apperror.Forbidden("无权创建根部门")
	}
	ok, err := s.hasDepartmentAccess(ctx, actor, *req.ParentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("无权在目标上级部门下创建")
	}
	return s.Create(ctx, req)
}

func (s *DepartmentService) Update(ctx context.Context, id uint, req DepartmentUpdateRequest) (*DepartmentDetailResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("部门不存在")
		}
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if name == "" || code == "" {
		return nil, apperror.BadRequest("部门名称和编码不能为空")
	}
	if req.ParentID != nil && *req.ParentID == id {
		return nil, apperror.BadRequest("上级部门不能选择自己")
	}
	if err := s.ensureLeaderExists(ctx, req.LeaderID); err != nil {
		return nil, err
	}
	if exists, err := s.repo.ExistsByCode(ctx, code, id); err != nil {
		return nil, err
	} else if exists {
		return nil, apperror.Conflict("部门编码已存在")
	}
	if exists, err := s.repo.ExistsByNameInParent(ctx, req.ParentID, name, id); err != nil {
		return nil, err
	} else if exists {
		return nil, apperror.Conflict("同级部门名称已存在")
	}

	oldParentID := item.ParentID
	oldAncestors := item.Ancestors
	oldLevel := item.Level
	newAncestors, newLevel, err := s.resolveAncestorsAndLevel(ctx, req.ParentID)
	if err != nil {
		return nil, err
	}
	if req.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *req.ParentID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, apperror.BadRequest("上级部门不存在")
			}
			return nil, err
		}
		selfPath := fmt.Sprintf("%s%d/", oldAncestors, item.ID)
		if strings.HasPrefix(parent.Ancestors, selfPath) {
			return nil, apperror.BadRequest("不能将部门移动到其子级部门下")
		}
	}

	item.ParentID = req.ParentID
	item.Name = name
	item.Code = code
	item.Ancestors = newAncestors
	item.Level = newLevel
	item.Sort = req.Sort
	item.Status = req.Status
	item.LeaderID = req.LeaderID
	item.Phone = strings.TrimSpace(req.Phone)
	item.Email = strings.ToLower(strings.TrimSpace(req.Email))
	item.Remark = strings.TrimSpace(req.Remark)

	err = s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(item).Error; err != nil {
			return err
		}
		if sameUintPtr(oldParentID, req.ParentID) && oldAncestors == newAncestors && oldLevel == newLevel {
			return nil
		}
		oldPrefix := fmt.Sprintf("%s%d/", oldAncestors, item.ID)
		newPrefix := fmt.Sprintf("%s%d/", newAncestors, item.ID)
		levelDiff := newLevel - oldLevel
		if err := tx.Model(&model.Department{}).
			Where("ancestors LIKE ?", oldPrefix+"%").
			Update("ancestors", gorm.Expr("REPLACE(ancestors, ?, ?)", oldPrefix, newPrefix)).Error; err != nil {
			return err
		}
		if levelDiff != 0 {
			if err := tx.Model(&model.Department{}).
				Where("ancestors LIKE ?", oldPrefix+"%").
				Update("level", gorm.Expr("level + ?", levelDiff)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(ctx, *item)
	return &resp, nil
}

func (s *DepartmentService) UpdateByActor(ctx context.Context, actor *auth.CurrentUser, id uint, req DepartmentUpdateRequest) (*DepartmentDetailResponse, error) {
	if actor == nil {
		return nil, apperror.Unauthorized("未登录或登录已失效")
	}
	if !isSuperAdmin(actor.RoleCodes) {
		ok, err := s.hasDepartmentAccess(ctx, actor, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apperror.Forbidden("无访问权限")
		}
		if req.ParentID != nil {
			ok, err = s.hasDepartmentAccess(ctx, actor, *req.ParentID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, apperror.Forbidden("无权迁移到目标上级部门")
			}
		}
	}
	return s.Update(ctx, id, req)
}

func (s *DepartmentService) Delete(ctx context.Context, id uint) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperror.NotFound("部门不存在")
		}
		return err
	}
	children, err := s.repo.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if children > 0 {
		return apperror.BadRequest("请先删除子部门后再删除当前部门")
	}
	users, err := s.repo.CountUsers(ctx, id)
	if err != nil {
		return err
	}
	if users > 0 {
		return apperror.BadRequest("该部门下仍有关联用户，请先调整用户归属")
	}
	return s.repo.DeleteByID(ctx, item.ID)
}

func (s *DepartmentService) DeleteByActor(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if actor == nil {
		return apperror.Unauthorized("未登录或登录已失效")
	}
	if !isSuperAdmin(actor.RoleCodes) {
		ok, err := s.hasDepartmentAccess(ctx, actor, id)
		if err != nil {
			return err
		}
		if !ok {
			return apperror.Forbidden("无访问权限")
		}
	}
	return s.Delete(ctx, id)
}

func (s *DepartmentService) resolveAncestorsAndLevel(ctx context.Context, parentID *uint) (string, int, error) {
	if parentID == nil {
		return "/", 1, nil
	}
	parent, err := s.repo.GetByID(ctx, *parentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", 0, apperror.BadRequest("上级部门不存在")
		}
		return "", 0, err
	}
	ancestors := fmt.Sprintf("%s%d/", parent.Ancestors, parent.ID)
	return ancestors, parent.Level + 1, nil
}

func (s *DepartmentService) ensureLeaderExists(ctx context.Context, leaderID *uint) error {
	if leaderID == nil || *leaderID == 0 {
		return nil
	}
	_, err := s.userRepo.GetByID(ctx, *leaderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperror.BadRequest("部门负责人不存在")
		}
		return err
	}
	return nil
}

func (s *DepartmentService) buildLeaderMap(ctx context.Context, list []model.Department) (map[uint]string, error) {
	ids := make([]uint, 0)
	seen := make(map[uint]struct{})
	for _, item := range list {
		if item.LeaderID == nil || *item.LeaderID == 0 {
			continue
		}
		if _, ok := seen[*item.LeaderID]; ok {
			continue
		}
		ids = append(ids, *item.LeaderID)
		seen[*item.LeaderID] = struct{}{}
	}
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(users))
	for _, u := range users {
		result[u.ID] = u.Nickname
	}
	return result, nil
}

func (s *DepartmentService) buildUserCountMap(ctx context.Context) (map[uint]int64, error) {
	type row struct {
		DepartmentID uint
		Count        int64
	}
	rows := make([]row, 0)
	err := s.repo.DB().WithContext(ctx).
		Model(&model.User{}).
		Select("department_id, COUNT(1) AS count").
		Where("department_id IS NOT NULL").
		Group("department_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]int64, len(rows))
	for _, r := range rows {
		result[r.DepartmentID] = r.Count
	}
	return result, nil
}

func (s *DepartmentService) toResponse(ctx context.Context, item model.Department) DepartmentDetailResponse {
	var leaderName string
	if item.LeaderID != nil && *item.LeaderID > 0 {
		if u, err := s.userRepo.GetByID(ctx, *item.LeaderID); err == nil {
			leaderName = u.Nickname
		}
	}
	var userCount int64
	if c, err := s.repo.CountUsers(ctx, item.ID); err == nil {
		userCount = c
	}
	return DepartmentDetailResponse{
		ID:         item.ID,
		ParentID:   item.ParentID,
		Name:       item.Name,
		Code:       item.Code,
		Ancestors:  item.Ancestors,
		Level:      item.Level,
		Sort:       item.Sort,
		Status:     item.Status,
		LeaderID:   item.LeaderID,
		LeaderName: leaderName,
		Phone:      item.Phone,
		Email:      item.Email,
		Remark:     item.Remark,
		UserCount:  userCount,
		CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func buildDepartmentTree(all []model.Department, leaderMap map[uint]string, userCount map[uint]int64) []DepartmentDetailResponse {
	nodes := make(map[uint]*DepartmentDetailResponse, len(all))
	roots := make([]*DepartmentDetailResponse, 0)
	for _, item := range all {
		leaderName := ""
		if item.LeaderID != nil {
			leaderName = leaderMap[*item.LeaderID]
		}
		node := &DepartmentDetailResponse{
			ID:         item.ID,
			ParentID:   item.ParentID,
			Name:       item.Name,
			Code:       item.Code,
			Ancestors:  item.Ancestors,
			Level:      item.Level,
			Sort:       item.Sort,
			Status:     item.Status,
			LeaderID:   item.LeaderID,
			LeaderName: leaderName,
			Phone:      item.Phone,
			Email:      item.Email,
			Remark:     item.Remark,
			UserCount:  userCount[item.ID],
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  item.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		nodes[item.ID] = node
	}
	for _, item := range all {
		node := nodes[item.ID]
		if item.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[*item.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, *node)
	}
	sortDepartmentChildren(roots)
	result := make([]DepartmentDetailResponse, 0, len(roots))
	for _, r := range roots {
		result = append(result, *r)
	}
	return result
}

func sortDepartmentChildren(nodes []*DepartmentDetailResponse) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Sort != nodes[j].Sort {
			return nodes[i].Sort < nodes[j].Sort
		}
		return nodes[i].ID < nodes[j].ID
	})
	for _, n := range nodes {
		if len(n.Children) == 0 {
			continue
		}
		inner := make([]*DepartmentDetailResponse, 0, len(n.Children))
		for idx := range n.Children {
			inner = append(inner, &n.Children[idx])
		}
		sortDepartmentChildren(inner)
	}
}

func sameUintPtr(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
