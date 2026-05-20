package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
)

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
		return nil, svcerr.Pass(ctx, "project", "ListServices", err)
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
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgac7e51a53391)
			}
			return nil, svcerr.Pass(ctx, "project", "UpsertService", err)
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
			return nil, svcerr.Pass(ctx, "project", "UpsertService", err)
		}
	} else {
		if err := s.serviceRepo.Save(ctx, it); err != nil {
			return nil, svcerr.Pass(ctx, "project", "UpsertService", err)
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
		return nil, svcerr.Pass(ctx, "project", "ListLogSources", err)
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
				return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg9d63941807e2)
			}
			return nil, svcerr.Pass(ctx, "project", "UpsertLogSource", err)
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
			return nil, svcerr.Pass(ctx, "project", "UpsertLogSource", err)
		}
	} else {
		if err := s.logRepo.Save(ctx, it); err != nil {
			return nil, svcerr.Pass(ctx, "project", "UpsertLogSource", err)
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
	AfterID     uint64  `form:"after_id"`
	Include     *string `form:"include"`
	Exclude     *string `form:"exclude"`
	Highlight   *string `form:"highlight"`
	FilePath    *string `form:"file_path"`
}

type logStreamPlan struct{}

// BuildLogStreamPlan 构建相关的业务逻辑。
func (s *ProjectMgmtService) BuildLogStreamPlan(ctx context.Context, q LogStreamQuery) (*logStreamPlan, error) {
	return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgb399afd1b3b2)
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
	return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg36453c419629)
}

// ListRemoteLogUnits 查询列表相关的业务逻辑。
func (s *ProjectMgmtService) ListRemoteLogUnits(ctx context.Context, q RemoteLogUnitQuery) ([]string, error) {
	return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg255ca1122356)
}

// ValidateLogSourceAccess 校验日志源属于项目下指定服务器（SSE/导出/审计共用）。
func (s *ProjectMgmtService) ValidateLogSourceAccess(ctx context.Context, projectID, serverID, logSourceID uint) error {
	if projectID == 0 {
		return constants.ErrProjectIDRequired
	}
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrLogSourceServerNotFound
		}
		return svcerr.Pass(ctx, "project", "ValidateLogSourceAccess", err)
	}
	if sv.ProjectID != projectID {
		return constants.ErrServerNotInCurrentProject
	}
	ok, err := s.logRepo.BelongsToProjectServer(ctx, projectID, serverID, logSourceID)
	if err != nil {
		return svcerr.Pass(ctx, "project", "ValidateLogSourceAccess", err)
	}
	if !ok {
		return constants.ErrNotFoundWithMsg(constants.ErrMsg9d63941807e2)
	}
	return nil
}

// ExportLogs 导出相关的业务逻辑。
func (s *ProjectMgmtService) ExportLogs(ctx context.Context, q LogExportQuery) ([]byte, string, error) {
	if err := s.ValidateLogSourceAccess(ctx, q.ProjectID, q.ServerID, q.LogSourceID); err != nil {
		return nil, "", svcerr.Pass(ctx, "project", "ExportLogs", err)
	}

	var includeRe *regexp.Regexp
	var err error
	if q.Include != nil && strings.TrimSpace(*q.Include) != "" {
		includeRe, err = regexp.Compile(strings.TrimSpace(*q.Include))
		if err != nil {
			return nil, "", constants.ErrBadRequestWithMsg(constants.ErrMsg1e7f0cdb6585)
		}
	}
	var excludeRe *regexp.Regexp
	if q.Exclude != nil && strings.TrimSpace(*q.Exclude) != "" {
		excludeRe, err = regexp.Compile(strings.TrimSpace(*q.Exclude))
		if err != nil {
			return nil, "", constants.ErrBadRequestWithMsg(constants.ErrMsg9bbaf0815790)
		}
	}

	maxLines := q.MaxLines
	if maxLines <= 0 {
		maxLines = 2000
	}
	if maxLines > maxLogHistoryPerStream {
		maxLines = maxLogHistoryPerStream
	}
	key := BuildLogStreamKey(q.ProjectID, q.ServerID, q.LogSourceID)
	events := AgentLogBroker.Snapshot(key, maxLines)
	lines := make([]string, 0, len(events))
	for _, ev := range events {
		line := strings.TrimSpace(ev.Line)
		if line == "" {
			continue
		}
		if includeRe != nil && !includeRe.MatchString(line) {
			continue
		}
		if excludeRe != nil && excludeRe.MatchString(line) {
			continue
		}
		if fp := strings.TrimSpace(ev.FilePath); fp != "" {
			lines = append(lines, fmt.Sprintf("[%s] %s", fp, line))
		} else {
			lines = append(lines, line)
		}
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	filename := fmt.Sprintf("project-%d-server-%d-source-%d-logs-%s.txt",
		q.ProjectID, q.ServerID, q.LogSourceID, time.Now().Format("20060102-150405"))
	return []byte(content), filename, nil
}
