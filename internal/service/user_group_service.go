package service

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type UserGroupItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	MemberCount int64  `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
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
		ID:          g.ID,
		Name:        g.Name,
		Code:        g.Code,
		Description: g.Description,
		Status:      g.Status,
		MemberCount: memberCount,
		CreatedAt:   g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   g.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type UserGroupService struct {
	repo     *repository.UserGroupRepository
	userRepo *repository.UserRepository
}

func NewUserGroupService(repo *repository.UserGroupRepository, userRepo *repository.UserRepository) *UserGroupService {
	return &UserGroupService{repo: repo, userRepo: userRepo}
}

func (s *UserGroupService) List(ctx context.Context, query UserGroupListQuery) (*pagination.Result[UserGroupItem], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, repository.UserGroupListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
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
		return nil, err
	}
	n, _ := s.repo.CountMembers(ctx, id)
	ids, err := s.repo.ListMemberUserIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	members := make([]UserGroupMemberRow, 0)
	if len(ids) > 0 && s.userRepo != nil {
		users, err := s.userRepo.ListByIDs(ctx, ids)
		if err != nil {
			return nil, err
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
		return nil, err
	}
	st := req.Status
	if st != model.StatusDisabled {
		st = model.StatusEnabled
	}
	g := &model.UserGroup{
		Name:        name,
		Code:        code,
		Description: strings.TrimSpace(req.Description),
		Status:      st,
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, err
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
		return nil, err
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
	if err := s.repo.Save(ctx, g); err != nil {
		return nil, err
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
		return err
	}
	if err := s.repo.ReplaceMemberUserIDs(ctx, id, nil); err != nil {
		return err
	}
	return s.repo.Delete(ctx, g)
}

func (s *UserGroupService) AssignUsers(ctx context.Context, id uint, req UserGroupAssignUsersRequest) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserGroupNotFound
		}
		return err
	}
	ids := dedupeUints(req.UserIDs)
	if len(ids) > 0 {
		if s.userRepo == nil {
			return constants.ErrInternal
		}
		users, err := s.userRepo.ListByIDs(ctx, ids)
		if err != nil {
			return err
		}
		if len(users) != len(ids) {
			return constants.ErrBadRequestWithMsg("存在无效的用户 ID")
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
