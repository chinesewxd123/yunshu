package service

import (
	"context"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/repository"

	"github.com/xuri/excelize/v2"
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
	encryptionKey string,
) (*ProjectMgmtService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
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
		aead:             aead,
		ensuredProjectAt: make(map[uint]time.Time),
	}, nil
}

type ProjectItem struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
	Status      int     `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

func toProjectItem(p model.Project) ProjectItem {
	return ProjectItem{
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Description: p.Description,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}
}

type ProjectListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListProjects 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListProjects(ctx context.Context, q ProjectListQuery) (*pagination.Result[ProjectItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.projectRepo.List(ctx, repository.ProjectListParams{Keyword: strings.TrimSpace(q.Keyword), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	out := make([]ProjectItem, 0, len(list))
	for _, it := range list {
		out = append(out, toProjectItem(it))
	}
	return &pagination.Result[ProjectItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type ProjectCreateRequest struct {
	Name        string  `json:"name" binding:"required,max=128"`
	Code        string  `json:"code" binding:"required,max=64"`
	Description *string `json:"description"`
	Status      int     `json:"status"`
}

// CreateProject 创建项目；creatorUserID>0 时自动将创建人写入 project_members 为 owner。
func (s *ProjectMgmtService) CreateProject(ctx context.Context, creatorUserID uint, req ProjectCreateRequest) (*ProjectItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	p := model.Project{Name: strings.TrimSpace(req.Name), Code: strings.TrimSpace(req.Code), Description: req.Description, Status: status}
	if err := s.projectRepo.Create(ctx, &p); err != nil {
		return nil, err
	}
	if s.memberRepo != nil && creatorUserID > 0 {
		m := model.ProjectMember{ProjectID: p.ID, UserID: creatorUserID, Role: "owner"}
		if err := s.memberRepo.Create(ctx, &m); err != nil {
			_ = s.projectRepo.DeleteByID(ctx, p.ID)
			return nil, fmt.Errorf("项目已创建但写入负责人失败，请重试或联系管理员: %w", err)
		}
	}
	item := toProjectItem(p)
	return &item, nil
}

type ProjectUpdateRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Description *string `json:"description"`
	Status      *int    `json:"status"`
}

// UpdateProject 更新相关的业务逻辑。
func (s *ProjectMgmtService) UpdateProject(ctx context.Context, id uint, req ProjectUpdateRequest) (*ProjectItem, error) {
	p, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("项目不存在")
		}
		return nil, err
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
	if err := s.projectRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	item := toProjectItem(*p)
	return &item, nil
}

// DeleteProject 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteProject(ctx context.Context, id uint) error {
	if s.memberRepo != nil {
		_ = s.memberRepo.DeleteByProject(ctx, id)
	}
	if err := s.projectRepo.DeleteByID(ctx, id); err != nil {
		return err
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
			return nil, apperror.NotFound("项目不存在")
		}
		return nil, err
	}
	rows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, err
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
			return nil, apperror.NotFound("项目不存在")
		}
		return nil, err
	}
	if s.userRepo == nil {
		return nil, apperror.Internal("用户仓储未初始化")
	}
	if _, err := s.userRepo.GetByID(ctx, req.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("用户不存在")
		}
		return nil, err
	}
	if _, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, req.UserID); err == nil {
		return nil, apperror.BadRequest("该用户已在项目中")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row := model.ProjectMember{
		ProjectID: projectID,
		UserID:    req.UserID,
		Role:      normalizeProjectMemberRole(req.Role),
	}
	if err := s.memberRepo.Create(ctx, &row); err != nil {
		return nil, err
	}
	drows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range drows {
		if r.ID == row.ID {
			it := toProjectMemberItems([]repository.ProjectMemberListRow{r})
			return &it[0], nil
		}
	}
	return nil, apperror.Internal("成员创建后查询失败")
}

// ProjectMemberUpdateRequest 更新成员角色。
type ProjectMemberUpdateRequest struct {
	Role string `json:"role" binding:"required,max=32"`
}

// UpdateProjectMember 更新项目内角色。
func (s *ProjectMgmtService) UpdateProjectMember(ctx context.Context, projectID, memberID uint, req ProjectMemberUpdateRequest) (*ProjectMemberItem, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("项目不存在")
		}
		return nil, err
	}
	m, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("成员记录不存在")
		}
		return nil, err
	}
	if m.ProjectID != projectID {
		return nil, apperror.BadRequest("成员不属于该项目")
	}
	m.Role = normalizeProjectMemberRole(req.Role)
	if err := s.memberRepo.Save(ctx, m); err != nil {
		return nil, err
	}
	drows, err := s.memberRepo.ListDisplayByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range drows {
		if r.ID == memberID {
			it := toProjectMemberItems([]repository.ProjectMemberListRow{r})
			return &it[0], nil
		}
	}
	return nil, apperror.Internal("成员更新后查询失败")
}

// RemoveProjectMember 移除项目成员。
func (s *ProjectMgmtService) RemoveProjectMember(ctx context.Context, projectID, memberID uint) error {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("项目不存在")
		}
		return err
	}
	m, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("成员记录不存在")
		}
		return err
	}
	if m.ProjectID != projectID {
		return apperror.BadRequest("成员不属于该项目")
	}
	return s.memberRepo.DeleteByID(ctx, memberID)
}

type ServerItem struct {
	ID                     uint    `json:"id"`
	ProjectID              uint    `json:"project_id"`
	GroupID                *uint   `json:"group_id,omitempty"`
	Name                   string  `json:"name"`
	Host                   string  `json:"host"`
	Port                   int     `json:"port"`
	OSType                 string  `json:"os_type"`
	OSArch                 string  `json:"os_arch"`
	Tags                   string  `json:"tags"`
	SourceType             string  `json:"source_type"`
	Provider               string  `json:"provider"`
	CloudInstanceID        string  `json:"cloud_instance_id"`
	CloudRegion            string  `json:"cloud_region"`
	CloudZone              string  `json:"cloud_zone"`
	CloudSpec              string  `json:"cloud_spec"`
	CloudConfigInfo        string  `json:"cloud_config_info"`
	CloudOSName            string  `json:"cloud_os_name"`
	CloudNetworkInfo       string  `json:"cloud_network_info"`
	CloudChargeType        string  `json:"cloud_charge_type"`
	CloudNetworkChargeType string  `json:"cloud_network_charge_type"`
	CloudTagsJSON          string  `json:"cloud_tags_json"`
	CloudPublicIP          string  `json:"cloud_public_ip"`
	CloudPrivateIP         string  `json:"cloud_private_ip"`
	CloudStatusText        string  `json:"cloud_status_text"`
	LastTestAt             *string `json:"last_test_at"`
	LastTestErr            *string `json:"last_test_error"`
	CreatedAt              string  `json:"created_at"`
	LastSeenAt             *string `json:"last_seen_at"`
	Status                 int     `json:"status"`
}

func toServerItem(sv model.Server) ServerItem {
	var lastTestAt *string
	if sv.LastTestAt != nil {
		x := sv.LastTestAt.Format(time.RFC3339)
		lastTestAt = &x
	}
	var lastSeenAt *string
	if sv.LastSeenAt != nil {
		x := sv.LastSeenAt.Format(time.RFC3339)
		lastSeenAt = &x
	}
	return ServerItem{
		ID:                     sv.ID,
		ProjectID:              sv.ProjectID,
		GroupID:                sv.GroupID,
		Name:                   sv.Name,
		Host:                   sv.Host,
		Port:                   sv.Port,
		OSType:                 sv.OSType,
		OSArch:                 sv.OSArch,
		Tags:                   sv.Tags,
		SourceType:             sv.SourceType,
		Provider:               sv.Provider,
		CloudInstanceID:        sv.CloudInstanceID,
		CloudRegion:            sv.CloudRegion,
		CloudZone:              sv.CloudZone,
		CloudSpec:              sv.CloudSpec,
		CloudConfigInfo:        sv.CloudConfigInfo,
		CloudOSName:            sv.CloudOSName,
		CloudNetworkInfo:       sv.CloudNetworkInfo,
		CloudChargeType:        sv.CloudChargeType,
		CloudNetworkChargeType: sv.CloudNetworkChargeType,
		CloudTagsJSON:          sv.CloudTagsJSON,
		CloudPublicIP:          sv.CloudPublicIP,
		CloudPrivateIP:         sv.CloudPrivateIP,
		CloudStatusText:        sv.CloudStatusText,
		LastTestAt:             lastTestAt,
		LastTestErr:            sv.LastTestError,
		CreatedAt:              sv.CreatedAt.Format(time.RFC3339),
		LastSeenAt:             lastSeenAt,
		Status:                 sv.Status,
	}
}

// ServerDetailItem 在 ServerItem 基础上附带 SSH 凭据元数据（不含密钥明文）。
type ServerDetailItem struct {
	ServerItem
	AuthType          string  `json:"auth_type,omitempty"`
	Username          string  `json:"username,omitempty"`
	PasswordSet       bool    `json:"password_set"`
	PrivateKeySet     bool    `json:"private_key_set"`
	UsernameDictLabel *string `json:"username_dict_label,omitempty"`
	PasswordDictLabel *string `json:"password_dict_label,omitempty"`
}

type ServerListQuery struct {
	ProjectID      uint   `form:"project_id" binding:"required"`
	Keyword        string `form:"keyword"`
	GroupID        *uint  `form:"group_id"`
	CloudAccountID *uint  `form:"cloud_account_id"`
	SourceType     string `form:"source_type"`
	Provider       string `form:"provider"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
}

// ListServers 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListServers(ctx context.Context, q ServerListQuery) (*pagination.Result[ServerItem], error) {
	if err := s.ensureDefaultServerGroups(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	// If cloud_account_id provided, resolve to its group_id for server filtering.
	groupID := q.GroupID
	if q.CloudAccountID != nil {
		acc, err := s.cloudAccountRepo.GetByID(ctx, *q.CloudAccountID)
		if err != nil {
			return nil, err
		}
		if acc.ProjectID != q.ProjectID {
			return nil, apperror.BadRequest("云账号不属于当前项目")
		}
		gid := acc.GroupID
		groupID = &gid
	}

	list, total, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID:  q.ProjectID,
		Keyword:    strings.TrimSpace(q.Keyword),
		GroupID:    groupID,
		SourceType: strings.TrimSpace(q.SourceType),
		Provider:   strings.TrimSpace(q.Provider),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ServerItem, 0, len(list))
	for _, it := range list {
		out = append(out, toServerItem(it))
	}
	return &pagination.Result[ServerItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type ServerUpsertRequest struct {
	ID              *uint  `json:"id"`
	ProjectID       uint   `json:"project_id" binding:"required"`
	GroupID         *uint  `json:"group_id,omitempty"`
	Name            string `json:"name" binding:"required"`
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port"`
	OSType          string `json:"os_type"`
	Tags            string `json:"tags"`
	Status          int    `json:"status"`
	SourceType      string `json:"source_type"`
	Provider        string `json:"provider"`
	CloudInstanceID string `json:"cloud_instance_id"`
	CloudRegion     string `json:"cloud_region"`

	AuthType   string  `json:"auth_type"` // password/key
	Username   string  `json:"username"`
	Password   *string `json:"password,omitempty"`
	PrivateKey *string `json:"private_key,omitempty"`
	Passphrase *string `json:"passphrase,omitempty"`

	UsernameDictLabel string `json:"username_dict_label"`
	PasswordDictLabel string `json:"password_dict_label"`
}

// UpsertServer 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertServer(ctx context.Context, req ServerUpsertRequest) (*ServerItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	if req.Port <= 0 {
		req.Port = 22
	}
	osType := strings.TrimSpace(req.OSType)
	if osType == "" {
		osType = "linux"
	}
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	var sv *model.Server
	var err error
	if req.ID != nil && *req.ID > 0 {
		sv, err = s.serverRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("服务器不存在")
			}
			return nil, err
		}
	} else {
		sv = &model.Server{}
	}

	sv.ProjectID = req.ProjectID
	sv.GroupID = req.GroupID
	sv.Name = strings.TrimSpace(req.Name)
	sv.Host = strings.TrimSpace(req.Host)
	sv.Port = req.Port
	sv.OSType = osType
	sv.Tags = strings.TrimSpace(req.Tags)
	sv.Status = status
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = model.ServerGroupCategorySelfHosted
	}
	sv.SourceType = sourceType
	sv.Provider = strings.TrimSpace(req.Provider)
	sv.CloudInstanceID = strings.TrimSpace(req.CloudInstanceID)
	sv.CloudRegion = strings.TrimSpace(req.CloudRegion)

	if sv.ID == 0 {
		if err := s.serverRepo.Create(ctx, sv); err != nil {
			return nil, err
		}
	} else {
		if err := s.serverRepo.Save(ctx, sv); err != nil {
			return nil, err
		}
	}

	// credential optional: only upsert when username provided
	if strings.TrimSpace(req.Username) != "" && strings.TrimSpace(req.AuthType) != "" {
		var existingCred *model.ServerCredential
		if sv.ID > 0 {
			c, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
			} else {
				existingCred = c
			}
		}
		cred, err := s.buildCredentialForSave(*sv, req, existingCred)
		if err != nil {
			return nil, err
		}
		if err := s.serverRepo.UpsertCredential(ctx, cred); err != nil {
			return nil, err
		}
	}

	item := toServerItem(*sv)
	return &item, nil
}

func (s *ProjectMgmtService) buildCredentialForSave(server model.Server, req ServerUpsertRequest, existing *model.ServerCredential) (*model.ServerCredential, error) {
	authType := strings.ToLower(strings.TrimSpace(req.AuthType))
	if authType == "" {
		authType = "password"
	}
	cred := &model.ServerCredential{ServerID: server.ID, AuthType: authType, Username: strings.TrimSpace(req.Username), KeyVersion: 1}
	if ul := strings.TrimSpace(req.UsernameDictLabel); ul != "" {
		cred.UsernameDictLabel = &ul
	}
	if pl := strings.TrimSpace(req.PasswordDictLabel); pl != "" {
		cred.PasswordDictLabel = &pl
	}
	switch authType {
	case "password":
		if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
			enc, err := cryptox.EncryptString(s.aead, *req.Password)
			if err != nil {
				return nil, err
			}
			cred.EncPassword = &enc
		} else if existing != nil && existing.EncPassword != nil && strings.TrimSpace(*existing.EncPassword) != "" {
			cred.EncPassword = existing.EncPassword
		} else {
			return nil, apperror.BadRequest("密码不能为空")
		}
	case "key":
		if req.PrivateKey != nil && strings.TrimSpace(*req.PrivateKey) != "" {
			encKey, err := cryptox.EncryptString(s.aead, *req.PrivateKey)
			if err != nil {
				return nil, err
			}
			cred.EncPrivateKey = &encKey
		} else if existing != nil && existing.EncPrivateKey != nil && strings.TrimSpace(*existing.EncPrivateKey) != "" {
			cred.EncPrivateKey = existing.EncPrivateKey
		} else {
			return nil, apperror.BadRequest("私钥不能为空")
		}
		if req.Passphrase != nil && strings.TrimSpace(*req.Passphrase) != "" {
			encPP, err := cryptox.EncryptString(s.aead, *req.Passphrase)
			if err != nil {
				return nil, err
			}
			cred.EncPassphrase = &encPP
		} else if existing != nil {
			cred.EncPassphrase = existing.EncPassphrase
		}
	default:
		return nil, apperror.BadRequest("认证类型不合法")
	}
	return cred, nil
}

// DeleteServer 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteServer(ctx context.Context, id uint) error {
	return s.serverRepo.DeleteByID(ctx, id)
}

// GetServer 获取相关的业务逻辑。
func (s *ProjectMgmtService) GetServer(ctx context.Context, id uint) (*ServerDetailItem, error) {
	sv, err := s.serverRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("服务器不存在")
		}
		return nil, err
	}
	base := toServerItem(*sv)
	out := &ServerDetailItem{ServerItem: base}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return out, nil
		}
		return nil, err
	}
	out.AuthType = cred.AuthType
	out.Username = cred.Username
	out.PasswordSet = cred.EncPassword != nil && strings.TrimSpace(*cred.EncPassword) != ""
	out.PrivateKeySet = cred.EncPrivateKey != nil && strings.TrimSpace(*cred.EncPrivateKey) != ""
	out.UsernameDictLabel = cloneStrPtr(cred.UsernameDictLabel)
	out.PasswordDictLabel = cloneStrPtr(cred.PasswordDictLabel)
	return out, nil
}

func cloneStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	cp := v
	return &cp
}

type ServerExecRequest struct {
	ProjectID  uint   `json:"project_id"`
	ServerID   uint   `json:"server_id"`
	Command    string `json:"command" binding:"required"`
	TimeoutSec int    `json:"timeout_sec"`
}

type ServerExecResult struct {
	ServerID   uint   `json:"server_id"`
	Command    string `json:"command"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Truncated  bool   `json:"truncated"`
}

// ExecServerCommand 执行对应的业务逻辑。
func (s *ProjectMgmtService) ExecServerCommand(ctx context.Context, req ServerExecRequest) (*ServerExecResult, error) {
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("服务器不存在")
		}
		return nil, err
	}
	if sv.ProjectID != req.ProjectID {
		return nil, apperror.BadRequest("服务器不属于当前项目")
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.BadRequest("服务器凭据未配置")
		}
		return nil, err
	}

	sshCfg, err := s.decryptCredentialToSSHConfig(*sv, *cred)
	if err != nil {
		return nil, err
	}

	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	if timeoutSec > 120 {
		timeoutSec = 120
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cli, err := sshclient.Dial(cctx, sshCfg)
	if err != nil {
		return nil, apperror.BadRequest("ssh connect failed: " + err.Error())
	}
	defer cli.Close()

	cmd := strings.TrimSpace(req.Command)
	result, err := cli.Exec(cctx, cmd, 256*1024)
	if err != nil {
		return nil, apperror.BadRequest("command exec failed: " + err.Error())
	}
	return &ServerExecResult{
		ServerID:   sv.ID,
		Command:    cmd,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMS: result.Duration.Milliseconds(),
		Truncated:  result.Truncated,
	}, nil
}

// StreamServerTerminal 执行对应的业务逻辑。
func (s *ProjectMgmtService) StreamServerTerminal(ctx context.Context, projectID, serverID uint, stdin io.Reader, stdout, stderr io.Writer, sizes <-chan sshclient.TerminalSize) error {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("服务器不存在")
		}
		return err
	}
	if sv.ProjectID != projectID {
		return apperror.BadRequest("服务器不属于当前项目")
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.BadRequest("服务器凭据未配置")
		}
		return err
	}
	sshCfg, err := s.decryptCredentialToSSHConfig(*sv, *cred)
	if err != nil {
		return err
	}
	cli, err := sshclient.Dial(ctx, sshCfg)
	if err != nil {
		return apperror.BadRequest("ssh connect failed: " + err.Error())
	}
	defer cli.Close()
	return cli.ShellStream(ctx, stdin, stdout, stderr, sizes)
}

type ServerGroupItem struct {
	ID        uint              `json:"id"`
	ProjectID uint              `json:"project_id"`
	ParentID  *uint             `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Category  string            `json:"category"`
	Provider  string            `json:"provider"`
	Sort      int               `json:"sort"`
	Status    int               `json:"status"`
	Children  []ServerGroupItem `json:"children,omitempty"`
}

type ServerGroupUpsertRequest struct {
	ID        *uint  `json:"id,omitempty"`
	ProjectID uint   `json:"project_id" binding:"required"`
	ParentID  *uint  `json:"parent_id,omitempty"`
	Name      string `json:"name" binding:"required"`
	Category  string `json:"category"`
	Provider  string `json:"provider"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status"`
}

type ServerGroupTreeQuery struct {
	ProjectID uint `form:"project_id" binding:"required"`
}

// ListServerGroupTree 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListServerGroupTree(ctx context.Context, q ServerGroupTreeQuery) ([]ServerGroupItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	list, err := s.serverGroupRepo.ListByProject(ctx, q.ProjectID)
	if err != nil {
		return nil, err
	}
	items := make([]ServerGroupItem, 0, len(list))
	for _, it := range list {
		items = append(items, ServerGroupItem{
			ID: it.ID, ProjectID: it.ProjectID, ParentID: it.ParentID, Name: it.Name,
			Category: it.Category, Provider: it.Provider, Sort: it.Sort, Status: it.Status,
		})
	}
	byParent := map[uint][]ServerGroupItem{}
	roots := make([]ServerGroupItem, 0)
	for _, it := range items {
		if it.ParentID == nil {
			roots = append(roots, it)
			continue
		}
		byParent[*it.ParentID] = append(byParent[*it.ParentID], it)
	}
	var attach func(*ServerGroupItem)
	attach = func(node *ServerGroupItem) {
		children := byParent[node.ID]
		for i := range children {
			child := children[i]
			attach(&child)
			node.Children = append(node.Children, child)
		}
	}
	for i := range roots {
		attach(&roots[i])
	}
	return roots, nil
}

// UpsertServerGroup 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertServerGroup(ctx context.Context, req ServerGroupUpsertRequest) (*ServerGroupItem, error) {
	if err := s.ensureDefaultServerGroups(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = model.ServerGroupCategorySelfHosted
	}
	provider := strings.TrimSpace(req.Provider)
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}

	var item *model.ServerGroup
	var err error
	if req.ID != nil && *req.ID > 0 {
		item, err = s.serverGroupRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("服务器分组不存在")
			}
			return nil, err
		}
	} else {
		item = &model.ServerGroup{}
	}
	item.ProjectID = req.ProjectID
	item.ParentID = req.ParentID
	item.Name = strings.TrimSpace(req.Name)
	item.Category = category
	item.Provider = provider
	item.Sort = req.Sort
	item.Status = status
	if item.ID == 0 {
		err = s.serverGroupRepo.Create(ctx, item)
	} else {
		err = s.serverGroupRepo.Save(ctx, item)
	}
	if err != nil {
		return nil, err
	}
	return &ServerGroupItem{
		ID: item.ID, ProjectID: item.ProjectID, ParentID: item.ParentID, Name: item.Name,
		Category: item.Category, Provider: item.Provider, Sort: item.Sort, Status: item.Status,
	}, nil
}

// DeleteServerGroup 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteServerGroup(ctx context.Context, projectID, groupID uint) error {
	group, err := s.serverGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("服务器分组不存在")
		}
		return err
	}
	if group.ProjectID != projectID {
		return apperror.BadRequest("分组不属于当前项目")
	}
	return s.serverGroupRepo.DeleteByID(ctx, groupID)
}

type CloudAccountItem struct {
	ID            uint    `json:"id"`
	ProjectID     uint    `json:"project_id"`
	GroupID       uint    `json:"group_id"`
	Provider      string  `json:"provider"`
	AccountName   string  `json:"account_name"`
	RegionScope   string  `json:"region_scope"`
	Status        int     `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	LastSyncError *string `json:"last_sync_error,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type CloudAccountListQuery struct {
	ProjectID uint  `form:"project_id" binding:"required"`
	GroupID   *uint `form:"group_id"`
}

type CloudAccountUpsertRequest struct {
	ID          *uint  `json:"id,omitempty"`
	ProjectID   uint   `json:"project_id" binding:"required"`
	GroupID     uint   `json:"group_id" binding:"required"`
	Provider    string `json:"provider" binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	RegionScope string `json:"region_scope"`
	AK          string `json:"ak,omitempty"`
	SK          string `json:"sk,omitempty"`
	Status      int    `json:"status"`
}

func toCloudAccountItem(it model.CloudAccount) CloudAccountItem {
	var lastSyncAt *string
	if it.LastSyncAt != nil {
		v := it.LastSyncAt.Format(time.RFC3339)
		lastSyncAt = &v
	}
	return CloudAccountItem{
		ID: it.ID, ProjectID: it.ProjectID, GroupID: it.GroupID, Provider: it.Provider,
		AccountName: it.AccountName, RegionScope: it.RegionScope, Status: it.Status,
		LastSyncAt: lastSyncAt, LastSyncError: it.LastSyncError, CreatedAt: it.CreatedAt.Format(time.RFC3339),
	}
}

// ListCloudAccounts 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListCloudAccounts(ctx context.Context, q CloudAccountListQuery) ([]CloudAccountItem, error) {
	list, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, q.ProjectID, q.GroupID)
	if err != nil {
		return nil, err
	}
	out := make([]CloudAccountItem, 0, len(list))
	for _, it := range list {
		out = append(out, toCloudAccountItem(it))
	}
	return out, nil
}

// UpsertCloudAccount 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertCloudAccount(ctx context.Context, req CloudAccountUpsertRequest) (*CloudAccountItem, error) {
	var item *model.CloudAccount
	var err error
	if req.ID != nil && *req.ID > 0 {
		item, err = s.cloudAccountRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("云账号不存在")
			}
			return nil, err
		}
	} else {
		item = &model.CloudAccount{}
	}

	item.ProjectID = req.ProjectID
	item.GroupID = req.GroupID
	item.Provider = strings.TrimSpace(req.Provider)
	item.AccountName = strings.TrimSpace(req.AccountName)
	item.RegionScope = strings.TrimSpace(req.RegionScope)
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	item.Status = status

	// 校验同一项目下是否已有相同 AK 的云账号，避免重复添加
	if strings.TrimSpace(req.AK) != "" {
		accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, req.ProjectID, nil)
		if err != nil {
			return nil, err
		}
		for _, ex := range accounts {
			if req.ID != nil && ex.ID == *req.ID {
				continue
			}
			if ex.EncAK == nil {
				continue
			}
			dec, derr := cryptox.DecryptString(s.aead, *ex.EncAK)
			if derr != nil {
				continue
			}
			if strings.TrimSpace(dec) == strings.TrimSpace(req.AK) {
				return nil, apperror.BadRequest("相同 AK 的云账号已存在")
			}
		}
	}

	if strings.TrimSpace(req.AK) != "" {
		encAK, err := cryptox.EncryptString(s.aead, req.AK)
		if err != nil {
			return nil, err
		}
		item.EncAK = &encAK
	}
	if strings.TrimSpace(req.SK) != "" {
		encSK, err := cryptox.EncryptString(s.aead, req.SK)
		if err != nil {
			return nil, err
		}
		item.EncSK = &encSK
	}
	if item.ID == 0 {
		err = s.cloudAccountRepo.Create(ctx, item)
	} else {
		err = s.cloudAccountRepo.Save(ctx, item)
	}
	if err != nil {
		return nil, err
	}
	out := toCloudAccountItem(*item)
	return &out, nil
}

type CloudSyncRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
	AccountID uint `json:"account_id" binding:"required"`
}

type CloudSyncResult struct {
	Total     int    `json:"total"`
	Added     int    `json:"added"`
	Updated   int    `json:"updated"`
	Disabled  int    `json:"disabled"`
	Unchanged int    `json:"unchanged"`
	Message   string `json:"message"`
}

type CloudInstance struct {
	InstanceID        string
	Name              string
	Host              string
	Region            string
	Zone              string
	Spec              string
	ConfigInfo        string
	OSName            string
	NetworkInfo       string
	ChargeType        string
	NetworkChargeType string
	TagsJSON          string
	PublicIP          string
	PrivateIP         string
	StatusText        string
	OSType            string
	Status            int
}

type CloudProvider interface {
	ListInstances(ctx context.Context, ak, sk, regionScope string) ([]CloudInstance, error)
	QueryInstanceExpireAt(ctx context.Context, ak, sk, region, instanceID string) (*time.Time, error)
	ResetInstancePassword(ctx context.Context, ak, sk, region, instanceID, newPassword string) error
	RebootInstance(ctx context.Context, ak, sk, region, instanceID string) error
	ShutdownInstance(ctx context.Context, ak, sk, region, instanceID string) error
	SyncInstanceTags(ctx context.Context, ak, sk, region, instanceID string, oldTags, newTags map[string]string) error
}

func (s *ProjectMgmtService) providerFor(name string) (CloudProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alibaba", "aliyun":
		return &AlibabaCloudProvider{}, nil
	case "tencent", "qcloud":
		return &TencentCloudProvider{}, nil
	case "jd", "jingdong":
		return &JdCloudProvider{}, nil
	default:
		return nil, apperror.BadRequest("不支持的云厂商")
	}
}

// SyncCloudAccount 同步相关的业务逻辑。
func (s *ProjectMgmtService) SyncCloudAccount(ctx context.Context, req CloudSyncRequest) (*CloudSyncResult, error) {
	acc, err := s.cloudAccountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("云账号不存在")
		}
		return nil, err
	}
	if acc.ProjectID != req.ProjectID {
		return nil, apperror.BadRequest("云账号不属于当前项目")
	}
	if acc.EncAK == nil || acc.EncSK == nil {
		return nil, apperror.BadRequest("AK/SK 未配置")
	}
	ak, err := cryptox.DecryptString(s.aead, *acc.EncAK)
	if err != nil {
		return nil, err
	}
	sk, err := cryptox.DecryptString(s.aead, *acc.EncSK)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerFor(acc.Provider)
	if err != nil {
		return nil, err
	}
	instances, err := provider.ListInstances(ctx, ak, sk, acc.RegionScope)
	now := time.Now()
	acc.LastSyncAt = &now
	if err != nil {
		msg := err.Error()
		acc.LastSyncError = &msg
		_ = s.cloudAccountRepo.Save(ctx, acc)
		return nil, err
	}
	acc.LastSyncError = nil
	_ = s.cloudAccountRepo.Save(ctx, acc)

	currentServers, _ := s.serverRepo.ListByProjectGroupProvider(ctx, acc.ProjectID, acc.GroupID, acc.Provider)
	allCloudServers, _ := s.serverRepo.ListByProjectProviderCloud(ctx, acc.ProjectID, acc.Provider)
	existedByInstanceID := make(map[string]*model.Server, len(allCloudServers))
	for i := range allCloudServers {
		instanceID := strings.TrimSpace(allCloudServers[i].CloudInstanceID)
		if instanceID == "" {
			continue
		}
		existedByInstanceID[instanceID] = &allCloudServers[i]
	}
	seen := make(map[string]struct{}, len(instances))
	added := 0
	updated := 0
	unchanged := 0
	for _, ins := range instances {
		instanceID := strings.TrimSpace(ins.InstanceID)
		if instanceID != "" {
			seen[instanceID] = struct{}{}
		}
		existed := existedByInstanceID[instanceID]
		var reqID *uint
		changed := true
		if existed != nil {
			reqID = &existed.ID
			changed = strings.TrimSpace(existed.Name) != strings.TrimSpace(ins.Name) ||
				strings.TrimSpace(existed.Host) != strings.TrimSpace(ins.Host) ||
				strings.TrimSpace(existed.CloudRegion) != strings.TrimSpace(ins.Region) ||
				strings.TrimSpace(existed.OSType) != strings.TrimSpace(ins.OSType) ||
				existed.Status != ins.Status ||
				existed.GroupID == nil || *existed.GroupID != acc.GroupID
		}
		groupID := acc.GroupID
		_, upErr := s.UpsertServer(ctx, ServerUpsertRequest{
			ID:              reqID,
			ProjectID:       acc.ProjectID,
			GroupID:         &groupID,
			Name:            ins.Name,
			Host:            ins.Host,
			Port:            22,
			OSType:          ins.OSType,
			Tags:            "cloud-sync",
			Status:          ins.Status,
			SourceType:      model.ServerGroupCategoryCloud,
			Provider:        acc.Provider,
			CloudInstanceID: instanceID,
			CloudRegion:     ins.Region,
		})
		if upErr == nil {
			if existed == nil {
				added++
			} else if changed {
				updated++
			} else {
				unchanged++
			}
		}
	}

	disabled := 0
	for _, sv := range currentServers {
		if strings.TrimSpace(sv.CloudInstanceID) == "" {
			continue
		}
		if _, ok := seen[sv.CloudInstanceID]; ok {
			continue
		}
		if sv.Status != model.StatusDisabled {
			sv.Status = model.StatusDisabled
			if err := s.serverRepo.Save(ctx, &sv); err == nil {
				disabled++
			}
		}
	}
	return &CloudSyncResult{
		Total: len(instances), Added: added, Updated: updated, Disabled: disabled, Unchanged: unchanged, Message: "sync completed",
	}, nil
}

// DeleteCloudAccount 删除云账号
func (s *ProjectMgmtService) DeleteCloudAccount(ctx context.Context, projectID, accountID uint) error {
	acc, err := s.cloudAccountRepo.GetByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("云账号不存在")
		}
		return err
	}
	if acc.ProjectID != projectID {
		return apperror.BadRequest("云账号不属于当前项目")
	}
	return s.cloudAccountRepo.DeleteByID(ctx, accountID)
}

func (s *ProjectMgmtService) ensureDefaultServerGroups(ctx context.Context, projectID uint) error {
	// Avoid repeated maintenance scans on hot list endpoints.
	s.ensureMu.Lock()
	if ts, ok := s.ensuredProjectAt[projectID]; ok && time.Since(ts) < 30*time.Second {
		s.ensureMu.Unlock()
		return nil
	}
	s.ensureMu.Unlock()

	list, err := s.serverGroupRepo.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		selfHosted := model.ServerGroup{
			ProjectID: projectID,
			Name:      "自建服务器",
			Category:  model.ServerGroupCategorySelfHosted,
			Provider:  "custom",
			Sort:      1,
			Status:    model.StatusEnabled,
		}
		cloudRoot := model.ServerGroup{
			ProjectID: projectID,
			Name:      "云服务器",
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "custom",
			Sort:      2,
			Status:    model.StatusEnabled,
		}
		if err := s.serverGroupRepo.Create(ctx, &selfHosted); err != nil {
			return err
		}
		if err := s.serverGroupRepo.Create(ctx, &cloudRoot); err != nil {
			return err
		}
		alibaba := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      "阿里云",
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "alibaba",
			Sort:      10,
			Status:    model.StatusEnabled,
		}
		tencent := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      "腾讯云",
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "tencent",
			Sort:      11,
			Status:    model.StatusEnabled,
		}
		jd := model.ServerGroup{
			ProjectID: projectID,
			ParentID:  &cloudRoot.ID,
			Name:      "京东云",
			Category:  model.ServerGroupCategoryCloud,
			Provider:  "jd",
			Sort:      12,
			Status:    model.StatusEnabled,
		}
		_ = s.serverGroupRepo.Create(ctx, &alibaba)
		_ = s.serverGroupRepo.Create(ctx, &tencent)
		_ = s.serverGroupRepo.Create(ctx, &jd)

	}

	// Backfill ungrouped servers every time (historical data compatibility).
	// This cannot rely on "len(list) == 0" only, because old rows may be inserted
	// later without group_id while default groups already exist.
	list, err = s.serverGroupRepo.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	var selfHostedID uint
	for _, g := range list {
		if g.ParentID != nil {
			continue
		}
		if strings.TrimSpace(g.Category) != model.ServerGroupCategorySelfHosted {
			continue
		}
		selfHostedID = g.ID
		break
	}
	if selfHostedID == 0 {
		// Fallback: create one self-hosted root if missing unexpectedly.
		selfHosted := model.ServerGroup{
			ProjectID: projectID,
			Name:      "自建服务器",
			Category:  model.ServerGroupCategorySelfHosted,
			Provider:  "custom",
			Sort:      1,
			Status:    model.StatusEnabled,
		}
		if err := s.serverGroupRepo.Create(ctx, &selfHosted); err != nil {
			return err
		}
		selfHostedID = selfHosted.ID
	}
	if servers, err := s.serverRepo.ListByProjectWithoutGroup(ctx, projectID); err == nil {
		for i := range servers {
			sv := servers[i]
			sv.GroupID = &selfHostedID
			if strings.TrimSpace(sv.SourceType) == "" {
				sv.SourceType = model.ServerGroupCategorySelfHosted
			}
			_ = s.serverRepo.Save(ctx, &sv)
		}
	}

	s.ensureMu.Lock()
	s.ensuredProjectAt[projectID] = time.Now()
	s.ensureMu.Unlock()
	return nil
}

type ServerTestRequest struct {
	ServerID uint `json:"server_id" binding:"required"`
}

type ServerTestResult struct {
	ServerID uint   `json:"server_id,omitempty"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
}

// TestServerConnectivity 测试相关的业务逻辑。
func (s *ProjectMgmtService) TestServerConnectivity(ctx context.Context, req ServerTestRequest) (*ServerTestResult, error) {
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sv.SourceType) == model.ServerGroupCategoryCloud {
		return s.testCloudServerConnectivityBySDK(ctx, sv)
	}
	return s.testSelfHostedServerConnectivityByTCP(ctx, sv)
}

type BatchServerTestRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	ServerIDs []uint `json:"server_ids"`
	Parallel  int    `json:"parallel"`
}

type BatchServerTestResult struct {
	Total   int                `json:"total"`
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
	Results []ServerTestResult `json:"results"`
}

type ServerSyncRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	ServerIDs []uint `json:"server_ids"`
	Parallel  int    `json:"parallel"`
}

type ServerSyncResult struct {
	Total       int                `json:"total"`
	Online      int                `json:"online"`
	Offline     int                `json:"offline"`
	UpdatedAt   string             `json:"updated_at"`
	TestResults []ServerTestResult `json:"test_results"`
}

// BatchTestServerConnectivity 执行对应的业务逻辑。
func (s *ProjectMgmtService) BatchTestServerConnectivity(ctx context.Context, req BatchServerTestRequest) (*BatchServerTestResult, error) {
	serverIDs, err := s.resolveProjectServerIDs(ctx, req.ProjectID, req.ServerIDs)
	if err != nil {
		return nil, err
	}
	if len(serverIDs) == 0 {
		return &BatchServerTestResult{Total: 0, Results: []ServerTestResult{}}, nil
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = 5
	}
	if parallel > 20 {
		parallel = 20
	}
	results := s.runServerConnectivityTests(ctx, serverIDs, parallel)
	out := &BatchServerTestResult{Total: len(results), Results: results}
	for _, r := range results {
		if r.OK {
			out.Success++
		} else {
			out.Failed++
		}
	}
	return out, nil
}

// SyncProjectServers 同步相关的业务逻辑。
func (s *ProjectMgmtService) SyncProjectServers(ctx context.Context, req ServerSyncRequest) (*ServerSyncResult, error) {
	serverIDs, err := s.resolveProjectServerIDs(ctx, req.ProjectID, req.ServerIDs)
	if err != nil {
		return nil, err
	}
	if len(serverIDs) == 0 {
		return &ServerSyncResult{UpdatedAt: time.Now().Format(time.RFC3339), TestResults: []ServerTestResult{}}, nil
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = 8
	}
	if parallel > 30 {
		parallel = 30
	}
	results := s.runServerConnectivityTests(ctx, serverIDs, parallel)
	out := &ServerSyncResult{
		Total:       len(results),
		UpdatedAt:   time.Now().Format(time.RFC3339),
		TestResults: results,
	}
	for _, r := range results {
		if r.OK {
			out.Online++
		} else {
			out.Offline++
		}
	}
	return out, nil
}

func (s *ProjectMgmtService) resolveProjectServerIDs(ctx context.Context, projectID uint, serverIDs []uint) ([]uint, error) {
	list, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: projectID,
		Page:      1,
		PageSize:  10000,
	})
	if err != nil {
		return nil, err
	}
	if len(serverIDs) == 0 {
		out := make([]uint, 0, len(list))
		for _, it := range list {
			out = append(out, it.ID)
		}
		return out, nil
	}
	allowed := make(map[uint]struct{}, len(list))
	for _, it := range list {
		allowed[it.ID] = struct{}{}
	}
	out := make([]uint, 0, len(serverIDs))
	for _, id := range serverIDs {
		if _, ok := allowed[id]; ok {
			out = append(out, id)
		}
	}
	return out, nil
}

func (s *ProjectMgmtService) runServerConnectivityTests(ctx context.Context, serverIDs []uint, parallel int) []ServerTestResult {
	type job struct {
		id uint
	}
	jobs := make(chan job, len(serverIDs))
	results := make(chan ServerTestResult, len(serverIDs))
	worker := func() {
		for it := range jobs {
			r, err := s.TestServerConnectivity(ctx, ServerTestRequest{ServerID: it.id})
			if err != nil {
				results <- ServerTestResult{ServerID: it.id, OK: false, Message: err.Error()}
				continue
			}
			results <- *r
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker()
		}()
	}
	for _, id := range serverIDs {
		jobs <- job{id: id}
	}
	close(jobs)
	wg.Wait()
	close(results)
	out := make([]ServerTestResult, 0, len(serverIDs))
	for it := range results {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerID < out[j].ServerID })
	return out
}

func (s *ProjectMgmtService) testSelfHostedServerConnectivityByTCP(ctx context.Context, sv *model.Server) (*ServerTestResult, error) {
	host := strings.TrimSpace(sv.Host)
	if host == "" {
		return nil, apperror.BadRequest("服务器地址不能为空")
	}
	port := sv.Port
	if port <= 0 {
		port = 22
	}
	target := fmt.Sprintf("%s:%d", host, port)
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(cctx, "tcp", target)
	now := time.Now()
	sv.LastTestAt = &now
	if err != nil {
		msg := "[TCP] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}
	_ = conn.Close()
	sv.LastTestError = nil
	_ = s.serverRepo.Save(ctx, sv)
	return &ServerTestResult{ServerID: sv.ID, OK: true, Message: fmt.Sprintf("[TCP] reachable: %s", target)}, nil
}

func (s *ProjectMgmtService) testCloudServerConnectivityBySDK(ctx context.Context, sv *model.Server) (*ServerTestResult, error) {
	now := time.Now()
	sv.LastTestAt = &now

	if sv.GroupID == nil {
		msg := "[SDK] cloud server missing group_id"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	groupID := *sv.GroupID
	accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, sv.ProjectID, &groupID)
	if err != nil {
		return nil, err
	}
	providerName := strings.TrimSpace(sv.Provider)
	var account *model.CloudAccount
	for i := range accounts {
		it := &accounts[i]
		if it.Status != model.StatusEnabled {
			continue
		}
		if providerName == "" || strings.EqualFold(strings.TrimSpace(it.Provider), providerName) {
			account = it
			break
		}
	}
	if account == nil {
		msg := "[SDK] no enabled cloud account found for this group/provider"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}
	if account.EncAK == nil || account.EncSK == nil {
		msg := "[SDK] cloud account AK/SK 未配置"
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	ak, err := cryptox.DecryptString(s.aead, *account.EncAK)
	if err != nil {
		return nil, err
	}
	sk, err := cryptox.DecryptString(s.aead, *account.EncSK)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerFor(account.Provider)
	if err != nil {
		msg := "[SDK] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	instances, err := provider.ListInstances(ctx, ak, sk, account.RegionScope)
	if err != nil {
		msg := "[SDK] " + err.Error()
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	instanceID := strings.TrimSpace(sv.CloudInstanceID)
	if instanceID == "" {
		// Best-effort: try to infer instance id by matching server host (public/private IP).
		host := strings.TrimSpace(sv.Host)
		for _, ins := range instances {
			if host == "" {
				break
			}
			if strings.EqualFold(strings.TrimSpace(ins.Host), host) ||
				strings.EqualFold(strings.TrimSpace(ins.PublicIP), host) ||
				strings.EqualFold(strings.TrimSpace(ins.PrivateIP), host) {
				instanceID = strings.TrimSpace(ins.InstanceID)
				if instanceID != "" {
					sv.CloudInstanceID = instanceID
					if strings.TrimSpace(sv.CloudRegion) == "" && strings.TrimSpace(ins.Region) != "" {
						sv.CloudRegion = strings.TrimSpace(ins.Region)
					}
					_ = s.serverRepo.Save(ctx, sv)
				}
				break
			}
		}
		if instanceID == "" {
			msg := "[SDK] cloud_instance_id 为空：请先通过「同步云账号」导入云服务器，或在服务器编辑里补充实例ID"
			sv.LastTestError = &msg
			_ = s.serverRepo.Save(ctx, sv)
			return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
		}
	}
	for _, ins := range instances {
		if strings.TrimSpace(ins.InstanceID) != instanceID {
			continue
		}
		if ins.Status == model.StatusEnabled {
			sv.LastTestError = nil
			_ = s.serverRepo.Save(ctx, sv)
			return &ServerTestResult{
				ServerID: sv.ID,
				OK:       true,
				Message:  fmt.Sprintf("[SDK] instance %s is running", instanceID),
			}, nil
		}
		msg := fmt.Sprintf("[SDK] instance %s is not running", instanceID)
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
	}

	msg := fmt.Sprintf("[SDK] instance %s not found in provider result", instanceID)
	sv.LastTestError = &msg
	_ = s.serverRepo.Save(ctx, sv)
	return &ServerTestResult{ServerID: sv.ID, OK: false, Message: msg}, nil
}

type CloudServerActionRequest struct {
	Action      string `json:"action" binding:"required,oneof=reset_password reboot shutdown"`
	NewPassword string `json:"new_password"`
}

type CloudServerActionResult struct {
	ServerID uint   `json:"server_id"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

func (s *ProjectMgmtService) RunCloudServerAction(ctx context.Context, projectID, serverID uint, req CloudServerActionRequest) (*CloudServerActionResult, error) {
	if strings.TrimSpace(req.Action) == "" {
		return nil, apperror.BadRequest("action 不能为空")
	}
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("服务器不存在")
		}
		return nil, err
	}
	if sv.ProjectID != projectID {
		return nil, apperror.BadRequest("服务器不属于当前项目")
	}
	if strings.TrimSpace(sv.SourceType) != model.ServerGroupCategoryCloud && strings.TrimSpace(sv.SourceType) != "cloud" {
		return nil, apperror.BadRequest("仅支持云服务器操作")
	}
	if sv.GroupID == nil {
		return nil, apperror.BadRequest("云服务器缺少 group_id")
	}
	groupID := *sv.GroupID
	accounts, err := s.cloudAccountRepo.ListByProjectAndGroup(ctx, sv.ProjectID, &groupID)
	if err != nil {
		return nil, err
	}
	providerName := strings.TrimSpace(sv.Provider)
	var account *model.CloudAccount
	for i := range accounts {
		it := &accounts[i]
		if it.Status != model.StatusEnabled {
			continue
		}
		if providerName == "" || strings.EqualFold(strings.TrimSpace(it.Provider), providerName) {
			account = it
			break
		}
	}
	if account == nil {
		return nil, apperror.BadRequest("未找到可用云账号（请先配置并同步云账号）")
	}
	if account.EncAK == nil || account.EncSK == nil {
		return nil, apperror.BadRequest("云账号 AK/SK 未配置")
	}
	ak, err := cryptox.DecryptString(s.aead, *account.EncAK)
	if err != nil {
		return nil, err
	}
	sk, err := cryptox.DecryptString(s.aead, *account.EncSK)
	if err != nil {
		return nil, err
	}
	provider, err := s.providerFor(account.Provider)
	if err != nil {
		return nil, err
	}

	instances, err := provider.ListInstances(ctx, ak, sk, account.RegionScope)
	if err != nil {
		return nil, apperror.BadRequest("[SDK] " + err.Error())
	}
	// Ensure instance id and region present.
	instanceID := strings.TrimSpace(sv.CloudInstanceID)
	region := strings.TrimSpace(sv.CloudRegion)
	if instanceID == "" || region == "" {
		host := strings.TrimSpace(sv.Host)
		for _, ins := range instances {
			if instanceID != "" && strings.EqualFold(strings.TrimSpace(ins.InstanceID), instanceID) {
				region = strings.TrimSpace(ins.Region)
				break
			}
			if instanceID == "" && host != "" &&
				(strings.EqualFold(strings.TrimSpace(ins.Host), host) ||
					strings.EqualFold(strings.TrimSpace(ins.PublicIP), host) ||
					strings.EqualFold(strings.TrimSpace(ins.PrivateIP), host)) {
				instanceID = strings.TrimSpace(ins.InstanceID)
				region = strings.TrimSpace(ins.Region)
				break
			}
		}
	}
	if instanceID == "" {
		return nil, apperror.BadRequest("cloud_instance_id 为空：请先通过「同步云账号」导入云服务器，或在服务器编辑里补充实例ID")
	}
	if region == "" {
		return nil, apperror.BadRequest("cloud_region 为空：请先同步云账号或在服务器编辑里补充地域")
	}
	// Persist inferred fields for later operations.
	changed := false
	if strings.TrimSpace(sv.CloudInstanceID) == "" && instanceID != "" {
		sv.CloudInstanceID = instanceID
		changed = true
	}
	if strings.TrimSpace(sv.CloudRegion) == "" && region != "" {
		sv.CloudRegion = region
		changed = true
	}
	if changed {
		_ = s.serverRepo.Save(ctx, sv)
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "reset_password":
		pw := strings.TrimSpace(req.NewPassword)
		if pw == "" {
			return nil, apperror.BadRequest("new_password 不能为空")
		}
		if err := provider.ResetInstancePassword(ctx, ak, sk, region, instanceID, pw); err != nil {
			return nil, apperror.BadRequest("[SDK] " + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "密码重置请求已提交"}, nil
	case "reboot":
		if err := provider.RebootInstance(ctx, ak, sk, region, instanceID); err != nil {
			return nil, apperror.BadRequest("[SDK] " + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "重启请求已提交"}, nil
	case "shutdown":
		if err := provider.ShutdownInstance(ctx, ak, sk, region, instanceID); err != nil {
			return nil, apperror.BadRequest("[SDK] " + err.Error())
		}
		return &CloudServerActionResult{ServerID: serverID, Action: action, Message: "关机请求已提交"}, nil
	default:
		return nil, apperror.BadRequest("不支持的 action")
	}
}

func (s *ProjectMgmtService) detectRemoteOSAndArch(ctx context.Context, cli *sshclient.Client) (string, string, string) {
	// Linux/macOS path
	if res, err := cli.Exec(ctx, "uname -s && uname -m && uname -a", 8192); err == nil && res.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
		osType := "linux"
		if len(lines) > 0 {
			v := strings.ToLower(strings.TrimSpace(lines[0]))
			if strings.Contains(v, "darwin") {
				osType = "darwin"
			} else if strings.Contains(v, "linux") {
				osType = "linux"
			}
		}
		arch := ""
		if len(lines) > 1 {
			arch = strings.TrimSpace(lines[1])
		}
		msg := strings.TrimSpace(res.Stdout)
		return osType, arch, msg
	}

	// Windows path (OpenSSH + powershell)
	psCmd := "powershell -NoProfile -Command \"[System.Runtime.InteropServices.RuntimeInformation]::OSDescription; [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture\""
	if res, err := cli.Exec(ctx, psCmd, 8192); err == nil && res.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
		arch := ""
		if len(lines) > 1 {
			arch = strings.TrimSpace(lines[len(lines)-1])
		}
		return "windows", arch, strings.TrimSpace(res.Stdout)
	}

	return "", "", "connected"
}

func (s *ProjectMgmtService) decryptCredentialToSSHConfig(sv model.Server, cred model.ServerCredential) (sshclient.Config, error) {
	cfg := sshclient.Config{
		Host:     sv.Host,
		Port:     sv.Port,
		Username: cred.Username,
	}
	switch strings.ToLower(strings.TrimSpace(cred.AuthType)) {
	case "password":
		if cred.EncPassword == nil {
			return sshclient.Config{}, apperror.BadRequest("缺少密码凭据")
		}
		pw, err := cryptox.DecryptString(s.aead, *cred.EncPassword)
		if err != nil {
			return sshclient.Config{}, err
		}
		cfg.AuthType = sshclient.AuthPassword
		cfg.Password = pw
	case "key":
		if cred.EncPrivateKey == nil {
			return sshclient.Config{}, apperror.BadRequest("缺少私钥凭据")
		}
		pk, err := cryptox.DecryptString(s.aead, *cred.EncPrivateKey)
		if err != nil {
			return sshclient.Config{}, err
		}
		cfg.AuthType = sshclient.AuthKey
		cfg.PrivateKey = pk
		if cred.EncPassphrase != nil {
			pp, err := cryptox.DecryptString(s.aead, *cred.EncPassphrase)
			if err != nil {
				return sshclient.Config{}, err
			}
			cfg.Passphrase = pp
		}
	default:
		return sshclient.Config{}, apperror.BadRequest("凭据认证类型不合法")
	}
	return cfg, nil
}

// ImportServersFromExcel expects first row header with:
// name,host,port,os_type,tags,auth_type,username,password,private_key,passphrase
func (s *ProjectMgmtService) ImportServersFromExcel(ctx context.Context, projectID uint, r io.Reader) (int, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return 0, apperror.BadRequest("Excel 文件不合法")
	}
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0, err
	}
	if len(rows) <= 1 {
		return 0, nil
	}
	header := map[string]int{}
	for i, h := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, key string) string {
		idx, ok := header[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	okCount := 0
	for _, row := range rows[1:] {
		name := get(row, "name")
		host := get(row, "host")
		if name == "" || host == "" {
			continue
		}
		port, _ := strconv.Atoi(get(row, "port"))
		if port <= 0 {
			port = 22
		}
		upReq := ServerUpsertRequest{
			ProjectID: projectID,
			Name:      name,
			Host:      host,
			Port:      port,
			OSType:    get(row, "os_type"),
			Tags:      get(row, "tags"),
			Status:    model.StatusEnabled,
			AuthType:  get(row, "auth_type"),
			Username:  get(row, "username"),
		}
		if pw := get(row, "password"); pw != "" {
			upReq.Password = &pw
		}
		if pk := get(row, "private_key"); pk != "" {
			upReq.PrivateKey = &pk
		}
		if pp := get(row, "passphrase"); pp != "" {
			upReq.Passphrase = &pp
		}
		if _, err := s.UpsertServer(ctx, upReq); err == nil {
			okCount++
		}
	}
	return okCount, nil
}

// ExportServersToExcel 导出相关的业务逻辑。
func (s *ProjectMgmtService) ExportServersToExcel(ctx context.Context, projectID uint, keyword string) (*excelize.File, error) {
	list, _, err := s.serverRepo.List(ctx, repository.ServerListParams{ProjectID: projectID, Keyword: strings.TrimSpace(keyword), Page: 1, PageSize: 10000})
	if err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"name", "host", "port", "os_type", "os_arch", "tags", "status", "last_test_at", "last_test_error"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for r, sv := range list {
		values := []any{sv.Name, sv.Host, sv.Port, sv.OSType, sv.OSArch, sv.Tags, sv.Status, "", ""}
		if sv.LastTestAt != nil {
			values[7] = sv.LastTestAt.Format(time.RFC3339)
		}
		if sv.LastTestError != nil {
			values[8] = *sv.LastTestError
		}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	return f, nil
}

// ServersImportTemplateExcel 执行对应的业务逻辑。
func (s *ProjectMgmtService) ServersImportTemplateExcel() (*excelize.File, error) {
	f := excelize.NewFile()

	// Template sheet
	sheet := f.GetSheetName(0)
	f.SetSheetName(sheet, "servers")
	sheet = "servers"

	headers := []string{"name", "host", "port", "os_type", "tags", "auth_type", "username", "password", "private_key", "passphrase"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// Example row (placeholders only)
	example := []any{
		"app-01",
		"10.0.0.12",
		22,
		"linux",
		"prod,web",
		"password",
		"root",
		"your_password",
		"",
		"",
	}
	for c, v := range example {
		cell, _ := excelize.CoordinatesToCellName(c+1, 2)
		_ = f.SetCellValue(sheet, cell, v)
	}

	// Notes sheet
	noteSheet := "说明"
	_, _ = f.NewSheet(noteSheet)
	notes := [][]any{
		{"字段", "说明", "必填", "示例/取值"},
		{"name", "服务器名称", "是", "app-01"},
		{"host", "主机地址/IP", "是", "10.0.0.12 / example.com"},
		{"port", "SSH 端口（默认 22）", "否", "22"},
		{"os_type", "操作系统类型（默认 linux）", "否", "linux / windows / darwin"},
		{"tags", "标签（逗号分隔）", "否", "prod,web"},
		{"auth_type", "认证方式（默认 password）", "否", "password / key"},
		{"username", "SSH 用户名", "否", "root"},
		{"password", "SSH 密码（auth_type=password 时可填）", "否", "your_password"},
		{"private_key", "SSH 私钥（auth_type=key 时可填，PEM 文本）", "否", "-----BEGIN ..."},
		{"passphrase", "私钥口令（可选）", "否", ""},
		{"注意", "表头必须保持为英文小写（与示例一致），导入时会按表头匹配列。空行会被跳过。", "", ""},
	}
	for r, row := range notes {
		cell, _ := excelize.CoordinatesToCellName(1, r+1)
		_ = f.SetSheetRow(noteSheet, cell, &row)
	}

	// Basic styling
	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 18)
	_ = f.SetColWidth(sheet, "C", "C", 8)
	_ = f.SetColWidth(sheet, "D", "D", 10)
	_ = f.SetColWidth(sheet, "E", "E", 18)
	_ = f.SetColWidth(sheet, "F", "F", 12)
	_ = f.SetColWidth(sheet, "G", "G", 12)
	_ = f.SetColWidth(sheet, "H", "H", 16)
	_ = f.SetColWidth(sheet, "I", "J", 18)

	_ = f.SetColWidth(noteSheet, "A", "A", 16)
	_ = f.SetColWidth(noteSheet, "B", "B", 48)
	_ = f.SetColWidth(noteSheet, "C", "C", 10)
	_ = f.SetColWidth(noteSheet, "D", "D", 26)

	f.SetActiveSheet(0)
	return f, nil
}

// sanity check: encryption key is in use
func (s *ProjectMgmtService) EncryptionReady() error {
	if s.aead == nil {
		return fmt.Errorf("encryption not ready")
	}
	return nil
}

type ServiceItem struct {
	ID        uint    `json:"id"`
	ServerID  uint    `json:"server_id"`
	Name      string  `json:"name"`
	Env       *string `json:"env"`
	Labels    *string `json:"labels"`
	Remark    *string `json:"remark"`
	Status    int     `json:"status"`
	CreatedAt string  `json:"created_at"`
}

func toServiceItem(it model.Service) ServiceItem {
	return ServiceItem{
		ID:        it.ID,
		ServerID:  it.ServerID,
		Name:      it.Name,
		Env:       it.Env,
		Labels:    it.Labels,
		Remark:    it.Remark,
		Status:    it.Status,
		CreatedAt: it.CreatedAt.Format(time.RFC3339),
	}
}

type ServiceListQuery struct {
	ProjectID uint   `form:"project_id" binding:"required"`
	ServerID  *uint  `form:"server_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

// ListServices 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListServices(ctx context.Context, q ServiceListQuery) (*pagination.Result[ServiceItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.serviceRepo.List(ctx, repository.ServiceListParams{
		ProjectID: q.ProjectID,
		ServerID:  q.ServerID,
		Keyword:   strings.TrimSpace(q.Keyword),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ServiceItem, 0, len(list))
	for _, it := range list {
		out = append(out, toServiceItem(it))
	}
	return &pagination.Result[ServiceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type ServiceUpsertRequest struct {
	ID       *uint   `json:"id"`
	ServerID uint    `json:"server_id" binding:"required"`
	Name     string  `json:"name" binding:"required"`
	Env      *string `json:"env"`
	Labels   *string `json:"labels"`
	Remark   *string `json:"remark"`
	Status   int     `json:"status"`
}

// UpsertService 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertService(ctx context.Context, req ServiceUpsertRequest) (*ServiceItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	var it *model.Service
	var err error
	if req.ID != nil && *req.ID > 0 {
		it, err = s.serviceRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("服务不存在")
			}
			return nil, err
		}
	} else {
		it = &model.Service{}
	}
	it.ServerID = req.ServerID
	it.Name = strings.TrimSpace(req.Name)
	it.Env = req.Env
	it.Labels = req.Labels
	it.Remark = req.Remark
	it.Status = status
	if it.ID == 0 {
		if err := s.serviceRepo.Create(ctx, it); err != nil {
			return nil, err
		}
	} else {
		if err := s.serviceRepo.Save(ctx, it); err != nil {
			return nil, err
		}
	}
	out := toServiceItem(*it)
	return &out, nil
}

// DeleteService 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteService(ctx context.Context, id uint) error {
	return s.serviceRepo.DeleteByID(ctx, id)
}

type LogSourceItem struct {
	ID            uint    `json:"id"`
	ServiceID     uint    `json:"service_id"`
	LogType       string  `json:"log_type"`
	Path          string  `json:"path"`
	Encoding      *string `json:"encoding"`
	Timezone      *string `json:"timezone"`
	MultilineRule *string `json:"multiline_rule"`
	IncludeRegex  *string `json:"include_regex"`
	ExcludeRegex  *string `json:"exclude_regex"`
	Status        int     `json:"status"`
	CreatedAt     string  `json:"created_at"`
}

func toLogSourceItem(it model.ServiceLogSource) LogSourceItem {
	return LogSourceItem{
		ID:            it.ID,
		ServiceID:     it.ServiceID,
		LogType:       it.LogType,
		Path:          it.Path,
		Encoding:      it.Encoding,
		Timezone:      it.Timezone,
		MultilineRule: it.MultilineRule,
		IncludeRegex:  it.IncludeRegex,
		ExcludeRegex:  it.ExcludeRegex,
		Status:        it.Status,
		CreatedAt:     it.CreatedAt.Format(time.RFC3339),
	}
}

type LogSourceListQuery struct {
	ProjectID uint  `form:"project_id" binding:"required"`
	ServiceID *uint `form:"service_id"`
	Page      int   `form:"page"`
	PageSize  int   `form:"page_size"`
}

// ListLogSources 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListLogSources(ctx context.Context, q LogSourceListQuery) (*pagination.Result[LogSourceItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.logRepo.List(ctx, repository.LogSourceListParams{ProjectID: q.ProjectID, ServiceID: q.ServiceID, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	out := make([]LogSourceItem, 0, len(list))
	for _, it := range list {
		out = append(out, toLogSourceItem(it))
	}
	return &pagination.Result[LogSourceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

type LogSourceUpsertRequest struct {
	ID            *uint   `json:"id"`
	ServiceID     uint    `json:"service_id" binding:"required"`
	LogType       string  `json:"log_type"`
	Path          string  `json:"path" binding:"required"`
	Encoding      *string `json:"encoding"`
	Timezone      *string `json:"timezone"`
	MultilineRule *string `json:"multiline_rule"`
	IncludeRegex  *string `json:"include_regex"`
	ExcludeRegex  *string `json:"exclude_regex"`
	Status        int     `json:"status"`
}

// UpsertLogSource 执行对应的业务逻辑。
func (s *ProjectMgmtService) UpsertLogSource(ctx context.Context, req LogSourceUpsertRequest) (*LogSourceItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	logType := strings.TrimSpace(req.LogType)
	if logType == "" {
		logType = "file"
	}
	var it *model.ServiceLogSource
	var err error
	if req.ID != nil && *req.ID > 0 {
		it, err = s.logRepo.GetByID(ctx, *req.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.NotFound("日志源不存在")
			}
			return nil, err
		}
	} else {
		it = &model.ServiceLogSource{}
	}
	it.ServiceID = req.ServiceID
	it.LogType = logType
	it.Path = strings.TrimSpace(req.Path)
	it.Encoding = req.Encoding
	it.Timezone = req.Timezone
	it.MultilineRule = req.MultilineRule
	it.IncludeRegex = req.IncludeRegex
	it.ExcludeRegex = req.ExcludeRegex
	it.Status = status
	if it.ID == 0 {
		if err := s.logRepo.Create(ctx, it); err != nil {
			return nil, err
		}
	} else {
		if err := s.logRepo.Save(ctx, it); err != nil {
			return nil, err
		}
	}
	out := toLogSourceItem(*it)
	return &out, nil
}

// DeleteLogSource 删除相关的业务逻辑。
func (s *ProjectMgmtService) DeleteLogSource(ctx context.Context, id uint) error {
	return s.logRepo.DeleteByID(ctx, id)
}

type LogStreamQuery struct {
	ProjectID   uint    `form:"project_id" binding:"required"`
	ServerID    uint    `form:"server_id" binding:"required"`
	LogSourceID uint    `form:"log_source_id" binding:"required"`
	TailLines   int     `form:"tail_lines"`
	Include     *string `form:"include"`
	Exclude     *string `form:"exclude"`
	Highlight   *string `form:"highlight"`
	FilePath    *string `form:"file_path"`
}

type logStreamPlan struct{}

// BuildLogStreamPlan 构建相关的业务逻辑。
func (s *ProjectMgmtService) BuildLogStreamPlan(ctx context.Context, q LogStreamQuery) (*logStreamPlan, error) {
	return nil, apperror.BadRequest("已移除 SSH 日志流，请使用 Agent 日志流")
}

type LogExportQuery struct {
	ProjectID   uint    `form:"project_id"`
	ServerID    uint    `form:"server_id" binding:"required"`
	LogSourceID uint    `form:"log_source_id" binding:"required"`
	MaxLines    int     `form:"max_lines"`
	Include     *string `form:"include"`
	Exclude     *string `form:"exclude"`
}

type RemoteLogFileQuery struct {
	ProjectID uint   `form:"project_id"`
	ServerID  uint   `form:"server_id" binding:"required"`
	Dir       string `form:"dir" binding:"required"`
}

type RemoteLogUnitQuery struct {
	ProjectID uint `form:"project_id"`
	ServerID  uint `form:"server_id" binding:"required"`
}

// ListRemoteLogFiles 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListRemoteLogFiles(ctx context.Context, q RemoteLogFileQuery) ([]string, error) {
	return nil, apperror.BadRequest("已移除 SSH 文件扫描，请手动配置路径并使用 Agent 模式")
}

// ListRemoteLogUnits 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListRemoteLogUnits(ctx context.Context, q RemoteLogUnitQuery) ([]string, error) {
	return nil, apperror.BadRequest("已移除 SSH 单元扫描，请手动配置 systemd 单元并使用 Agent 模式")
}

// ExportLogs 导出相关的业务逻辑。
func (s *ProjectMgmtService) ExportLogs(ctx context.Context, q LogExportQuery) ([]byte, string, error) {
	return nil, "", apperror.BadRequest("已移除 SSH 导出日志，仅支持 Agent 实时日志流")
}
