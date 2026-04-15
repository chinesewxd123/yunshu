package service

import (
	"context"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	cryptox "go-permission-system/internal/pkg/crypto"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/pkg/sshclient"
	"go-permission-system/internal/repository"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ProjectMgmtService struct {
	projectRepo *repository.ProjectRepository
	serverRepo  *repository.ServerRepository
	serviceRepo *repository.ServiceRepository
	logRepo     *repository.LogSourceRepository
	aead        cipher.AEAD
}

func NewProjectMgmtService(
	projectRepo *repository.ProjectRepository,
	serverRepo *repository.ServerRepository,
	serviceRepo *repository.ServiceRepository,
	logRepo *repository.LogSourceRepository,
	encryptionKey string,
) (*ProjectMgmtService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	return &ProjectMgmtService{projectRepo: projectRepo, serverRepo: serverRepo, serviceRepo: serviceRepo, logRepo: logRepo, aead: aead}, nil
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

func (s *ProjectMgmtService) CreateProject(ctx context.Context, req ProjectCreateRequest) (*ProjectItem, error) {
	status := req.Status
	if status != model.StatusDisabled {
		status = model.StatusEnabled
	}
	p := model.Project{Name: strings.TrimSpace(req.Name), Code: strings.TrimSpace(req.Code), Description: req.Description, Status: status}
	if err := s.projectRepo.Create(ctx, &p); err != nil {
		return nil, err
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

func (s *ProjectMgmtService) UpdateProject(ctx context.Context, id uint, req ProjectUpdateRequest) (*ProjectItem, error) {
	p, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("project not found")
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

func (s *ProjectMgmtService) DeleteProject(ctx context.Context, id uint) error {
	if err := s.projectRepo.DeleteByID(ctx, id); err != nil {
		return err
	}
	return nil
}

type ServerItem struct {
	ID          uint    `json:"id"`
	ProjectID   uint    `json:"project_id"`
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	OSType      string  `json:"os_type"`
	OSArch      string  `json:"os_arch"`
	Tags        string  `json:"tags"`
	LastTestAt  *string `json:"last_test_at"`
	LastTestErr *string `json:"last_test_error"`
	CreatedAt   string  `json:"created_at"`
	LastSeenAt  *string `json:"last_seen_at"`
	Status      int     `json:"status"`
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
		ID:          sv.ID,
		ProjectID:   sv.ProjectID,
		Name:        sv.Name,
		Host:        sv.Host,
		Port:        sv.Port,
		OSType:      sv.OSType,
		OSArch:      sv.OSArch,
		Tags:        sv.Tags,
		LastTestAt:  lastTestAt,
		LastTestErr: sv.LastTestError,
		CreatedAt:   sv.CreatedAt.Format(time.RFC3339),
		LastSeenAt:  lastSeenAt,
		Status:      sv.Status,
	}
}

type ServerListQuery struct {
	ProjectID uint   `form:"project_id" binding:"required"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (s *ProjectMgmtService) ListServers(ctx context.Context, q ServerListQuery) (*pagination.Result[ServerItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.serverRepo.List(ctx, repository.ServerListParams{ProjectID: q.ProjectID, Keyword: strings.TrimSpace(q.Keyword), Page: page, PageSize: pageSize})
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
	ID        *uint  `json:"id"`
	ProjectID uint   `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Host      string `json:"host" binding:"required"`
	Port      int    `json:"port"`
	OSType    string `json:"os_type"`
	Tags      string `json:"tags"`
	Status    int    `json:"status"`

	AuthType   string  `json:"auth_type"` // password/key
	Username   string  `json:"username"`
	Password   *string `json:"password,omitempty"`
	PrivateKey *string `json:"private_key,omitempty"`
	Passphrase *string `json:"passphrase,omitempty"`
}

func (s *ProjectMgmtService) UpsertServer(ctx context.Context, req ServerUpsertRequest) (*ServerItem, error) {
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
				return nil, apperror.NotFound("server not found")
			}
			return nil, err
		}
	} else {
		sv = &model.Server{}
	}

	sv.ProjectID = req.ProjectID
	sv.Name = strings.TrimSpace(req.Name)
	sv.Host = strings.TrimSpace(req.Host)
	sv.Port = req.Port
	sv.OSType = osType
	sv.Tags = strings.TrimSpace(req.Tags)
	sv.Status = status

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
		cred, err := s.buildCredentialForSave(*sv, req)
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

func (s *ProjectMgmtService) buildCredentialForSave(server model.Server, req ServerUpsertRequest) (*model.ServerCredential, error) {
	authType := strings.ToLower(strings.TrimSpace(req.AuthType))
	if authType == "" {
		authType = "password"
	}
	cred := &model.ServerCredential{ServerID: server.ID, AuthType: authType, Username: strings.TrimSpace(req.Username), KeyVersion: 1}
	switch authType {
	case "password":
		if req.Password == nil || strings.TrimSpace(*req.Password) == "" {
			return nil, apperror.BadRequest("password is required")
		}
		enc, err := cryptox.EncryptString(s.aead, *req.Password)
		if err != nil {
			return nil, err
		}
		cred.EncPassword = &enc
	case "key":
		if req.PrivateKey == nil || strings.TrimSpace(*req.PrivateKey) == "" {
			return nil, apperror.BadRequest("private_key is required")
		}
		encKey, err := cryptox.EncryptString(s.aead, *req.PrivateKey)
		if err != nil {
			return nil, err
		}
		cred.EncPrivateKey = &encKey
		if req.Passphrase != nil && strings.TrimSpace(*req.Passphrase) != "" {
			encPP, err := cryptox.EncryptString(s.aead, *req.Passphrase)
			if err != nil {
				return nil, err
			}
			cred.EncPassphrase = &encPP
		}
	default:
		return nil, apperror.BadRequest("invalid auth_type")
	}
	return cred, nil
}

func (s *ProjectMgmtService) DeleteServer(ctx context.Context, id uint) error {
	return s.serverRepo.DeleteByID(ctx, id)
}

type ServerTestRequest struct {
	ServerID uint `json:"server_id" binding:"required"`
}

type ServerTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *ProjectMgmtService) TestServerConnectivity(ctx context.Context, req ServerTestRequest) (*ServerTestResult, error) {
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		return nil, err
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, sv.ID)
	if err != nil {
		return nil, err
	}
	sshCfg, err := s.decryptCredentialToSSHConfig(*sv, *cred)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cli, err := sshclient.Dial(cctx, sshCfg)
	if err != nil {
		now := time.Now()
		msg := err.Error()
		sv.LastTestAt = &now
		sv.LastTestError = &msg
		_ = s.serverRepo.Save(ctx, sv)
		return &ServerTestResult{OK: false, Message: msg}, nil
	}
	defer cli.Close()
	detectedType, detectedArch, detectedMsg := s.detectRemoteOSAndArch(cctx, cli)
	now := time.Now()
	sv.LastTestAt = &now
	sv.LastTestError = nil
	if detectedType != "" {
		sv.OSType = detectedType
	}
	if detectedArch != "" {
		sv.OSArch = detectedArch
	}
	_ = s.serverRepo.Save(ctx, sv)
	msg := strings.TrimSpace(detectedMsg)
	if msg == "" {
		msg = "connected"
	}
	return &ServerTestResult{OK: true, Message: msg}, nil
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
			return sshclient.Config{}, apperror.BadRequest("missing password credential")
		}
		pw, err := cryptox.DecryptString(s.aead, *cred.EncPassword)
		if err != nil {
			return sshclient.Config{}, err
		}
		cfg.AuthType = sshclient.AuthPassword
		cfg.Password = pw
	case "key":
		if cred.EncPrivateKey == nil {
			return sshclient.Config{}, apperror.BadRequest("missing private key credential")
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
		return sshclient.Config{}, apperror.BadRequest("invalid credential auth_type")
	}
	return cfg, nil
}

// ImportServersFromExcel expects first row header with:
// name,host,port,os_type,tags,auth_type,username,password,private_key,passphrase
func (s *ProjectMgmtService) ImportServersFromExcel(ctx context.Context, projectID uint, r io.Reader) (int, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return 0, apperror.BadRequest("invalid excel file")
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
				return nil, apperror.NotFound("service not found")
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
				return nil, apperror.NotFound("log source not found")
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

func (s *ProjectMgmtService) BuildLogStreamPlan(ctx context.Context, q LogStreamQuery) (*logStreamPlan, error) {
	return nil, apperror.BadRequest("ssh log streaming removed; use agent log streaming")
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

func (s *ProjectMgmtService) ListRemoteLogFiles(ctx context.Context, q RemoteLogFileQuery) ([]string, error) {
	return nil, apperror.BadRequest("ssh file scan removed; configure path manually and use agent mode")
}

func (s *ProjectMgmtService) ListRemoteLogUnits(ctx context.Context, q RemoteLogUnitQuery) ([]string, error) {
	return nil, apperror.BadRequest("ssh unit scan removed; configure systemd unit manually and use agent mode")
}

func (s *ProjectMgmtService) ExportLogs(ctx context.Context, q LogExportQuery) ([]byte, string, error) {
	return nil, "", apperror.BadRequest("log export via ssh removed; only real-time agent stream is supported")
}
