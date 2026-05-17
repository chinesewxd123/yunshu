package service

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/projectaccess"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type UserGroupItem struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Code             string `json:"code"`
	Description      string `json:"description"`
	Status           int    `json:"status"`
	ScopeProjectID   *uint  `json:"scope_project_id,omitempty"`
	MemberCount      int64  `json:"member_count"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type UserGroupMemberRow struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type UserGroupDetailResponse struct {
	UserGroupItem
	Members []UserGroupMemberRow `json:"members"`
}

func NewUserGroupItem(g model.UserGroup, memberCount int64) UserGroupItem {
	return UserGroupItem{
		ID:             g.ID,
		Name:           g.Name,
		Code:           g.Code,
		Description:    g.Description,
		Status:         g.Status,
		ScopeProjectID: g.ScopeProjectID,
		MemberCount:    memberCount,
		CreatedAt:      g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type UserGroupService struct {
	repo         *repository.UserGroupRepository
	userRepo     *repository.UserRepository
	memberRepo   *repository.ProjectMemberRepository
	projectRepo  *repository.ProjectRepository
}

func NewUserGroupService(repo *repository.UserGroupRepository, userRepo *repository.UserRepository, memberRepo *repository.ProjectMemberRepository, projectRepo *repository.ProjectRepository) *UserGroupService {
	return &UserGroupService{repo: repo, userRepo: userRepo, memberRepo: memberRepo, projectRepo: projectRepo}
}

func (s *UserGroupService) ensureProjectAdmin(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrProjectMemberRequired
		}
		return svcerr.Pass("user-group", "ensureProjectAdmin", err)
	}
	if !projectaccess.RoleAtLeast(m.Role, "admin") {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

func (s *UserGroupService) ensureProjectMember(ctx context.Context, projectID, userID uint) error {
	if projectID == 0 || userID == 0 || s.memberRepo == nil {
		return nil
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, userID)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrBadRequestWithMsg("项目专属用户组的成员须为该项目成员")
	}
	return svcerr.Pass("user-group", "ensureProjectMember", err)
}

func (s *UserGroupService) ensureVisibleScopedGroup(ctx context.Context, g *model.UserGroup) error {
	if g == nil || g.ScopeProjectID == nil || *g.ScopeProjectID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, *g.ScopeProjectID, u.ID)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrForbidden
	}
	return svcerr.Pass("user-group", "ensureVisibleScopedGroup", err)
}

func (s *UserGroupService) List(ctx context.Context, query UserGroupListQuery) (*pagination.Result[UserGroupItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	var scope *uint
	if query.ScopeProjectID != nil && *query.ScopeProjectID > 0 {
		v := *query.ScopeProjectID
		scope = &v
	}
	list, total, err := s.repo.List(ctx, repository.UserGroupListParams{
		Keyword:        query.Keyword,
		Page:           page,
		PageSize:       pageSize,
		ScopeProjectID: scope,
	})
	if err != nil {
		return nil, svcerr.Pass("user-group", "List", err)
	}
	items := make([]UserGroupItem, 0, len(list))
	for i := range list {
		n, _ := s.repo.CountMembers(ctx, list[i].ID)
		items = append(items, NewUserGroupItem(list[i], n))
	}
	return &pagination.Result[UserGroupItem]{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *UserGroupService) Detail(ctx context.Context, id uint) (*UserGroupDetailResponse, error) {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserGroupNotFound
		}
		return nil, svcerr.Pass("user-group", "Detail", err)
	}
	if err := s.ensureVisibleScopedGroup(ctx, g); err != nil {
		return nil, svcerr.Pass("user-group", "Detail", err)
	}
	n, _ := s.repo.CountMembers(ctx, id)
	ids, err := s.repo.ListMemberUserIDs(ctx, id)
	if err != nil {
		return nil, svcerr.Pass("user-group", "Detail", err)
	}
	members := make([]UserGroupMemberRow, 0)
	if len(ids) > 0 && s.userRepo != nil {
		users, err := s.userRepo.ListByIDs(ctx, ids)
		if err != nil {
			return nil, svcerr.Pass("user-group", "Detail", err)
		}
		byID := make(map[uint]model.User, len(users))
		for _, u := range users {
			byID[u.ID] = u
		}
		for _, uid := range ids {
			if u, ok := byID[uid]; ok {
				members = append(members, UserGroupMemberRow{
					UserID:   u.ID,
					Username: u.Username,
					Nickname: u.Nickname,
				})
			}
		}
	}
	base := NewUserGroupItem(*g, n)
	return &UserGroupDetailResponse{UserGroupItem: base, Members: members}, nil
}

func (s *UserGroupService) Create(ctx context.Context, req UserGroupCreateRequest) (*UserGroupItem, error) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return nil, constants.ErrBadRequest
	}
	if _, err := s.repo.GetByCode(ctx, code); err == nil {
		return nil, constants.ErrConflictWithMsg("用户组编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, svcerr.Pass("user-group", "Create", err)
	}
	st := req.Status
	if st != model.StatusDisabled {
		st = model.StatusEnabled
	}
	var scope *uint
	if req.ScopeProjectID != nil && *req.ScopeProjectID > 0 {
		if s.projectRepo != nil {
			if _, err := s.projectRepo.GetByID(ctx, *req.ScopeProjectID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, constants.ErrProjectNotFound
				}
				return nil, svcerr.Pass("user-group", "Create", err)
			}
		}
		if err := s.ensureProjectAdmin(ctx, *req.ScopeProjectID); err != nil {
			return nil, svcerr.Pass("user-group", "Create", err)
		}
		v := *req.ScopeProjectID
		scope = &v
	}
	g := &model.UserGroup{
		Name:            name,
		Code:            code,
		Description:     strings.TrimSpace(req.Description),
		Status:          st,
		ScopeProjectID:  scope,
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, svcerr.Pass("user-group", "Create", err)
	}
	item := NewUserGroupItem(*g, 0)
	return &item, nil
}

func (s *UserGroupService) Update(ctx context.Context, id uint, req UserGroupUpdateRequest) (*UserGroupItem, error) {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserGroupNotFound
		}
		return nil, svcerr.Pass("user-group", "Update", err)
	}
	if err := s.ensureVisibleScopedGroup(ctx, g); err != nil {
		return nil, svcerr.Pass("user-group", "Update", err)
	}
	if req.Name != nil {
		g.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		g.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		g.Status = *req.Status
	}
	if req.ScopeProjectID != nil {
		if *req.ScopeProjectID == 0 {
			if g.ScopeProjectID != nil && *g.ScopeProjectID > 0 {
				if err := s.ensureProjectAdmin(ctx, *g.ScopeProjectID); err != nil {
					return nil, svcerr.Pass("user-group", "Update", err)
				}
			}
			g.ScopeProjectID = nil
		} else {
			if s.projectRepo != nil {
				if _, err := s.projectRepo.GetByID(ctx, *req.ScopeProjectID); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, constants.ErrProjectNotFound
					}
					return nil, svcerr.Pass("user-group", "Update", err)
				}
			}
			if err := s.ensureProjectAdmin(ctx, *req.ScopeProjectID); err != nil {
				return nil, svcerr.Pass("user-group", "Update", err)
			}
			v := *req.ScopeProjectID
			g.ScopeProjectID = &v
		}
	}
	if err := s.repo.Save(ctx, g); err != nil {
		return nil, svcerr.Pass("user-group", "Update", err)
	}
	n, _ := s.repo.CountMembers(ctx, id)
	item := NewUserGroupItem(*g, n)
	return &item, nil
}

func (s *UserGroupService) Delete(ctx context.Context, id uint) error {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserGroupNotFound
		}
		return svcerr.Pass("user-group", "Delete", err)
	}
	if err := s.ensureVisibleScopedGroup(ctx, g); err != nil {
		return svcerr.Pass("user-group", "Delete", err)
	}
	if err := s.repo.ReplaceMemberUserIDs(ctx, id, nil); err != nil {
		return svcerr.Pass("user-group", "Delete", err)
	}
	return s.repo.Delete(ctx, g)
}

func (s *UserGroupService) AssignUsers(ctx context.Context, id uint, req UserGroupAssignUsersRequest) error {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserGroupNotFound
		}
		return svcerr.Pass("user-group", "AssignUsers", err)
	}
	if err := s.ensureVisibleScopedGroup(ctx, g); err != nil {
		return svcerr.Pass("user-group", "AssignUsers", err)
	}
	ids := dedupeUints(req.UserIDs)
	if len(ids) > 0 {
		if s.userRepo == nil {
			return constants.ErrInternal
		}
		users, err := s.userRepo.ListByIDs(ctx, ids)
		if err != nil {
			return svcerr.Pass("user-group", "AssignUsers", err)
		}
		if len(users) != len(ids) {
			return constants.ErrBadRequestWithMsg("存在无效的用户 ID")
		}
		if g.ScopeProjectID != nil && *g.ScopeProjectID > 0 {
			for _, uid := range ids {
				if err := s.ensureProjectMember(ctx, *g.ScopeProjectID, uid); err != nil {
					return svcerr.Pass("user-group", "AssignUsers", err)
				}
			}
		}
	}
	return s.repo.ReplaceMemberUserIDs(ctx, id, ids)
}

func dedupeUints(in []uint) []uint {
	seen := map[uint]struct{}{}
	var out []uint
	for _, id := range in {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
