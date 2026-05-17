package service

import (
	"context"
	"crypto/cipher"
	"errors"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
)

type ProjectMgmtService struct {
	projectRepo      *repository.ProjectRepository
	serverRepo       *repository.ServerRepository
	serverGroupRepo  *repository.ServerGroupRepository
	cloudAccountRepo *repository.CloudAccountRepository
	serviceRepo      *repository.ServiceRepository
	logRepo          *repository.LogSourceRepository
	memberRepo       *repository.ProjectMemberRepository
	userRepo         *repository.UserRepository
	departmentRepo   *repository.DepartmentRepository
	aead             cipher.AEAD
	ensureMu         sync.Mutex
	ensuredProjectAt map[uint]time.Time
}

// NewProjectMgmtService 创建相关逻辑。
func NewProjectMgmtService(
	projectRepo *repository.ProjectRepository,
	serverRepo *repository.ServerRepository,
	serverGroupRepo *repository.ServerGroupRepository,
	cloudAccountRepo *repository.CloudAccountRepository,
	serviceRepo *repository.ServiceRepository,
	logRepo *repository.LogSourceRepository,
	memberRepo *repository.ProjectMemberRepository,
	userRepo *repository.UserRepository,
	departmentRepo *repository.DepartmentRepository,
	encryptionKey string,
) (*ProjectMgmtService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, svcerr.Pass("project", "NewProjectMgmtService", err)
	}
	return &ProjectMgmtService{
		projectRepo:      projectRepo,
		serverRepo:       serverRepo,
		serverGroupRepo:  serverGroupRepo,
		cloudAccountRepo: cloudAccountRepo,
		serviceRepo:      serviceRepo,
		logRepo:          logRepo,
		memberRepo:       memberRepo,
		userRepo:         userRepo,
		departmentRepo:   departmentRepo,
		aead:             aead,
		ensuredProjectAt: make(map[uint]time.Time),
	}, nil
}

type ProjectItem struct {
	ID                  uint    `json:"id"`
	Name                string  `json:"name"`
	Code                string  `json:"code"`
	Description         *string `json:"description"`
	Status              int     `json:"status"`
	OwnerDepartmentID   *uint   `json:"owner_department_id,omitempty"`
	// MyProjectRole 当前登录用户在该项目中的成员角色（owner/admin/member/readonly）；列表与更新接口在非超管时填充；超管可省略。
	MyProjectRole       string  `json:"my_project_role,omitempty"`
	CreatedAt           string  `json:"created_at"`
}

func toProjectItem(p model.Project) ProjectItem {
	return ProjectItem{
		ID:                p.ID,
		Name:              p.Name,
		Code:              p.Code,
		Description:       p.Description,
		Status:            p.Status,
		OwnerDepartmentID: p.OwnerDepartmentID,
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
	}
}

func (s *ProjectMgmtService) enrichMyProjectRole(ctx context.Context, item *ProjectItem) {
	if item == nil || s.memberRepo == nil {
		return
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) {
		return
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, item.ID, u.ID)
	if err != nil || m == nil {
		return
	}
	item.MyProjectRole = m.Role
}

func (s *ProjectMgmtService) enrichMyProjectRolesBatch(ctx context.Context, items []ProjectItem) {
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) || s.memberRepo == nil || len(items) == 0 {
		return
	}
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	roles, err := s.memberRepo.ListRolesByUserAndProjectIDs(ctx, u.ID, ids)
	if err != nil {
		return
	}
	for i := range items {
		if r, ok := roles[items[i].ID]; ok {
			items[i].MyProjectRole = r
		}
	}
}

type ProjectListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListProjects 查询列表相关的业务逻辑。非超级管理员仅能看到自己作为成员的项目。
func (s *ProjectMgmtService) ListProjects(ctx context.Context, q ProjectListQuery) (*pagination.Result[ProjectItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	params := repository.ProjectListParams{Keyword: strings.TrimSpace(q.Keyword), Page: page, PageSize: pageSize}
	var list []model.Project
	var total int64
	var err error
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) {
		list, total, err = s.projectRepo.ListVisibleToUser(ctx, u.ID, params)
	} else {
		list, total, err = s.projectRepo.List(ctx, params)
	}
	if err != nil {
		return nil, svcerr.Pass("project", "ListProjects", err)
	}
	out := make([]ProjectItem, 0, len(list))
	for _, it := range list {
		out = append(out, toProjectItem(it))
	}
	s.enrichMyProjectRolesBatch(ctx, out)
	return &pagination.Result[ProjectItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type ProjectCreateRequest struct {
	Name                string  `json:"name" binding:"required,max=128"`
	Code                string  `json:"code" binding:"required,max=64"`
	Description         *string `json:"description"`
	Status              int     `json:"status"`
	OwnerDepartmentID   *uint   `json:"owner_department_id"`
}

// CreateProject 创建项目；creatorUserID>0 时自动将创建人写入 project_members 为 owner。
func (s *ProjectMgmtService) CreateProject(ctx context.Context, creatorUserID uint, req ProjectCreateRequest) (*ProjectItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	if req.OwnerDepartmentID != nil && *req.OwnerDepartmentID > 0 && s.departmentRepo != nil {
		if _, err := s.departmentRepo.GetByID(ctx, *req.OwnerDepartmentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrDepartmentNotFound
			}
			return nil, svcerr.Pass("project", "CreateProject", err)
		}
	}
	var ownerDept *uint
	if req.OwnerDepartmentID != nil && *req.OwnerDepartmentID > 0 {
		v := *req.OwnerDepartmentID
		ownerDept = &v
	}
	p := model.Project{Name: strings.TrimSpace(req.Name), Code: strings.TrimSpace(req.Code), Description: req.Description, Status: status, OwnerDepartmentID: ownerDept}
	if err := s.projectRepo.Create(ctx, &p); err != nil {
		return nil, svcerr.Pass("project", "CreateProject", err)
	}
	if s.memberRepo != nil && creatorUserID > 0 {
		m := model.ProjectMember{ProjectID: p.ID, UserID: creatorUserID, Role: "owner"}
		if err := s.memberRepo.Create(ctx, &m); err != nil {
			_ = s.projectRepo.DeleteByID(ctx, p.ID)
			return nil, svcerr.Internal("project", "CreateProject", err, "项目已创建但写入负责人失败: %v")
		}
	}
	item := toProjectItem(p)
	if creatorUserID > 0 {
		item.MyProjectRole = "owner"
	}
	return &item, nil
}

type ProjectUpdateRequest struct {
	Name                *string `json:"name"`
	Code                *string `json:"code"`
	Description         *string `json:"description"`
	Status              *int    `json:"status"`
	OwnerDepartmentID   *uint   `json:"owner_department_id"`
}

// UpdateProject 更新相关的业务逻辑。
func (s *ProjectMgmtService) UpdateProject(ctx context.Context, id uint, req ProjectUpdateRequest) (*ProjectItem, error) {
	p, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrProjectNotFound
		}
		return nil, svcerr.Pass("project", "UpdateProject", err)
	}
	if req.Name != nil {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		p.Code = strings.TrimSpace(*req.Code)
	}
	if req.Description != nil {
		p.Description = req.Description
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.OwnerDepartmentID != nil {
		if *req.OwnerDepartmentID == 0 {
			p.OwnerDepartmentID = nil
		} else {
			if s.departmentRepo != nil {
				if _, err := s.departmentRepo.GetByID(ctx, *req.OwnerDepartmentID); err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, constants.ErrDepartmentNotFound
					}
					return nil, svcerr.Pass("project", "UpdateProject", err)
				}
			}
			v := *req.OwnerDepartmentID
			p.OwnerDepartmentID = &v
		}
	}
	if err := s.projectRepo.Save(ctx, p); err != nil {
		return nil, svcerr.Pass("project", "UpdateProject", err)
	}
	item := toProjectItem(*p)
	s.enrichMyProjectRole(ctx, &item)
	return &item, nil
}

// DeleteProject 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteProject(ctx context.Context, id uint) error {
	if s.memberRepo != nil {
		_ = s.memberRepo.DeleteByProject(ctx, id)
	}
	if err := s.projectRepo.DeleteByID(ctx, id); err != nil {
		return svcerr.Pass("project", "DeleteProject", err)
	}
	return nil
}

// --- 项目成员（project_members）：与项目资源、监控规则 project_id 形成租户闭环；成员邮箱并入规则通知（见 AlertRuleAssigneeService）。---

var allowedProjectMemberRoles = map[string]struct{}{
	"owner": {}, "admin": {}, "member": {}, "readonly": {},
}

func normalizeProjectMemberRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	if r == "" {
		return "member"
	}
	if _, ok := allowedProjectMemberRoles[r]; ok {
		return r
	}
	return "member"
}

// ProjectMemberItem 项目成员 API 展示。
type ProjectMemberItem struct {
	ID        uint    `json:"id"`
	UserID    uint    `json:"user_id"`
	Username  string  `json:"username"`
	Nickname  string  `json:"nickname"`
	Email     *string `json:"email"`
	Role      string  `json:"role"`
	CreatedAt string  `json:"created_at"`
}

func toProjectMemberItems(rows []repository.ProjectMemberListRow) []ProjectMemberItem {
	out := make([]ProjectMemberItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProjectMemberItem{
			ID:        r.ID,
			UserID:    r.UserID,
			Username:  r.Username,
			Nickname:  r.Nickname,
			Email:     r.Email,
			Role:      r.Role,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}
	return out
}

// ListProjectMembers 列出项目成员（含用户基本信息）。
func (s *ProjectMgmtService) ListProjectMembers(ctx context.Context, projectID uint) ([]ProjectMemberItem, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrProjectNotFound
		}
		return nil, svcerr.Pass("project", "ListProjectMembers", err)
	}
	rows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, svcerr.Pass("project", "ListProjectMembers", err)
	}
	return toProjectMemberItems(rows), nil
}

// ProjectMemberAddRequest 添加成员。
type ProjectMemberAddRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"omitempty,max=32"`
}

// AddProjectMember 将用户加入项目。
func (s *ProjectMgmtService) AddProjectMember(ctx context.Context, projectID uint, req ProjectMemberAddRequest) (*ProjectMemberItem, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrProjectNotFound
		}
		return nil, svcerr.Pass("project", "AddProjectMember", err)
	}
	if s.userRepo == nil {
		return nil, svcerr.InternalMsg("project", "api", constants.ErrMsgcc60c2c3c788)
	}
	if _, err := s.userRepo.GetByID(ctx, req.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, svcerr.Pass("project", "AddProjectMember", err)
	}
	if _, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, req.UserID); err == nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsga802e1b5e9e2)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, svcerr.Pass("project", "AddProjectMember", err)
	}
	row := model.ProjectMember{
		ProjectID: projectID,
		UserID:    req.UserID,
		Role:      normalizeProjectMemberRole(req.Role),
	}
	if err := s.memberRepo.Create(ctx, &row); err != nil {
		return nil, svcerr.Pass("project", "AddProjectMember", err)
	}
	drows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, svcerr.Pass("project", "AddProjectMember", err)
	}
	for _, r := range drows {
		if r.ID == row.ID {
			it := toProjectMemberItems([]repository.ProjectMemberListRow{r})
			return &it[0], nil
		}
	}
	return nil, svcerr.InternalMsg("project", "api", constants.ErrMsg1fe0209f952f)
}

// ProjectMemberUpdateRequest 更新成员角色。
type ProjectMemberUpdateRequest struct {
	Role string `json:"role" binding:"required,max=32"`
}

// UpdateProjectMember 更新项目内角色。
func (s *ProjectMgmtService) UpdateProjectMember(ctx context.Context, projectID, memberID uint, req ProjectMemberUpdateRequest) (*ProjectMemberItem, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrProjectNotFound
		}
		return nil, svcerr.Pass("project", "UpdateProjectMember", err)
	}
	m, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsge7773625bf8b)
		}
		return nil, svcerr.Pass("project", "UpdateProjectMember", err)
	}
	if m.ProjectID != projectID {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg461337ef3f89)
	}
	m.Role = normalizeProjectMemberRole(req.Role)
	if err := s.memberRepo.Save(ctx, m); err != nil {
		return nil, svcerr.Pass("project", "UpdateProjectMember", err)
	}
	drows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, svcerr.Pass("project", "UpdateProjectMember", err)
	}
	for _, r := range drows {
		if r.ID == memberID {
			it := toProjectMemberItems([]repository.ProjectMemberListRow{r})
			return &it[0], nil
		}
	}
	return nil, svcerr.InternalMsg("project", "api", constants.ErrMsg2940a3d4007c)
}

// RemoveProjectMember 移除项目成员。
func (s *ProjectMgmtService) RemoveProjectMember(ctx context.Context, projectID, memberID uint) error {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrProjectNotFound
		}
		return svcerr.Pass("project", "RemoveProjectMember", err)
	}
	m, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFoundWithMsg(constants.ErrMsge7773625bf8b)
		}
		return svcerr.Pass("project", "RemoveProjectMember", err)
	}
	if m.ProjectID != projectID {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg461337ef3f89)
	}
	return s.memberRepo.DeleteByID(ctx, memberID)
}

