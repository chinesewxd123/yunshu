package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/promapi"

	"gorm.io/gorm"
)

type AlertDatasourceListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type AlertDatasourceUpsertRequest struct {
	ProjectID         uint   `json:"project_id" binding:"required"`
	Name             string `json:"name" binding:"required,max=128"`
	Type             string `json:"type" binding:"omitempty,max=32"`
	BaseURL          string `json:"base_url" binding:"required,max=512"`
	BearerToken      string `json:"bearer_token"`
	BasicUser        string `json:"basic_user" binding:"omitempty,max=128"`
	BasicPassword    string `json:"basic_password" binding:"omitempty,max=256"`
	SkipTLSVerify    *bool  `json:"skip_tls_verify"`
	Enabled          *bool  `json:"enabled"`
	Remark           string `json:"remark" binding:"omitempty,max=512"`
	ClearBearerToken bool   `json:"clear_bearer_token"`
	ClearBasicAuth   bool   `json:"clear_basic_auth"`
}

type AlertDatasourceItem struct {
	model.AlertDatasource
	ProjectName string `json:"project_name,omitempty" gorm:"column:project_name"`
}

type PromQueryRequest struct {
	Query string `json:"query" binding:"required"`
	Time  string `json:"time"`
}

type PromQueryRangeRequest struct {
	Query string `json:"query" binding:"required"`
	Start string `json:"start" binding:"required"`
	End   string `json:"end" binding:"required"`
	Step  string `json:"step" binding:"required"`
}

type AlertDatasourceService struct {
	db *gorm.DB
}

func NewAlertDatasourceService(db *gorm.DB) *AlertDatasourceService {
	return &AlertDatasourceService{db: db}
}

func (s *AlertDatasourceService) mask(ds *model.AlertDatasource) {
	if strings.TrimSpace(ds.BearerToken) != "" {
		ds.BearerToken = "***"
	}
	if strings.TrimSpace(ds.BasicPassword) != "" {
		ds.BasicPassword = "***"
	}
}

func (s *AlertDatasourceService) List(ctx context.Context, q AlertDatasourceListQuery) ([]AlertDatasourceItem, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Table("alert_datasources ad").
		Select("ad.*, p.name AS project_name").
		Joins("LEFT JOIN projects p ON p.id = ad.project_id AND p.deleted_at IS NULL")
	if q.ProjectID > 0 {
		tx = tx.Where("ad.project_id = ?", q.ProjectID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("ad.name LIKE ? OR ad.base_url LIKE ? OR ad.remark LIKE ? OR p.name LIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	var list []AlertDatasourceItem
	if err := tx.Order("ad.id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, err
	}
	for i := range list {
		s.mask(&list[i].AlertDatasource)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDatasourceService) Get(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	var row model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("告警数据源不存在")
		}
		return nil, err
	}
	s.mask(&row)
	return &row, nil
}

func (s *AlertDatasourceService) getRaw(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	var row model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertDatasourceService) Create(ctx context.Context, req AlertDatasourceUpsertRequest) (*model.AlertDatasource, error) {
	t := strings.TrimSpace(req.Type)
	if t == "" {
		t = "prometheus"
	}
	if t != "prometheus" {
		return nil, apperror.BadRequest("暂仅支持 type=prometheus")
	}
	if req.ProjectID == 0 {
		return nil, apperror.BadRequest("project_id 必填")
	}
	row := model.AlertDatasource{
		ProjectID:     req.ProjectID,
		Name:          strings.TrimSpace(req.Name),
		Type:          t,
		BaseURL:       strings.TrimSpace(req.BaseURL),
		BearerToken:   strings.TrimSpace(req.BearerToken),
		BasicUser:     strings.TrimSpace(req.BasicUser),
		BasicPassword: strings.TrimSpace(req.BasicPassword),
		SkipTLSVerify: req.SkipTLSVerify != nil && *req.SkipTLSVerify,
		Enabled:       req.Enabled == nil || *req.Enabled,
		Remark:        strings.TrimSpace(req.Remark),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	s.mask(&row)
	return &row, nil
}

func (s *AlertDatasourceService) Update(ctx context.Context, id uint, req AlertDatasourceUpsertRequest) (*model.AlertDatasource, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("告警数据源不存在")
		}
		return nil, err
	}
	if req.ProjectID == 0 {
		return nil, apperror.BadRequest("project_id 必填")
	}
	if req.ProjectID != row.ProjectID {
		row.ProjectID = req.ProjectID
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.BaseURL) != "" {
		row.BaseURL = strings.TrimSpace(req.BaseURL)
	}
	if req.SkipTLSVerify != nil {
		row.SkipTLSVerify = *req.SkipTLSVerify
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	row.Remark = strings.TrimSpace(req.Remark)
	if req.ClearBearerToken {
		row.BearerToken = ""
	} else if strings.TrimSpace(req.BearerToken) != "" && req.BearerToken != "***" {
		row.BearerToken = strings.TrimSpace(req.BearerToken)
	}
	if req.ClearBasicAuth {
		row.BasicUser = ""
		row.BasicPassword = ""
	} else {
		if strings.TrimSpace(req.BasicUser) != "" {
			row.BasicUser = strings.TrimSpace(req.BasicUser)
		}
		if strings.TrimSpace(req.BasicPassword) != "" && req.BasicPassword != "***" {
			row.BasicPassword = strings.TrimSpace(req.BasicPassword)
		}
	}
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	s.mask(row)
	return row, nil
}

func (s *AlertDatasourceService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.AlertDatasource{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound("告警数据源不存在")
	}
	return nil
}

func (s *AlertDatasourceService) clientFor(ctx context.Context, id uint) (*promapi.Client, *model.AlertDatasource, error) {
	row, err := s.getRaw(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperror.NotFound("告警数据源不存在")
		}
		return nil, nil, err
	}
	if !row.Enabled {
		return nil, nil, apperror.BadRequest("数据源已停用")
	}
	if row.Type != "prometheus" {
		return nil, nil, apperror.BadRequest("仅 prometheus 数据源支持查询")
	}
	return &promapi.Client{
		BaseURL:       row.BaseURL,
		BearerToken:   row.BearerToken,
		BasicUser:     row.BasicUser,
		BasicPassword: row.BasicPassword,
		SkipTLSVerify: row.SkipTLSVerify,
	}, row, nil
}

// PromQuery 即时查询，返回 Prometheus 原始 JSON。
func (s *AlertDatasourceService) PromQuery(ctx context.Context, id uint, req PromQueryRequest) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, err
	}
	qctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	body, _, err := cli.QueryInstant(qctx, strings.TrimSpace(req.Query), strings.TrimSpace(req.Time))
	return body, err
}

// PromQueryRange 范围查询。
func (s *AlertDatasourceService) PromQueryRange(ctx context.Context, id uint, req PromQueryRangeRequest) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, err
	}
	qctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	body, _, err := cli.QueryRange(qctx, strings.TrimSpace(req.Query), strings.TrimSpace(req.Start), strings.TrimSpace(req.End), strings.TrimSpace(req.Step))
	return body, err
}

// PrometheusActiveAlerts 查询 Prometheus /api/v1/alerts（活跃告警快照）。
func (s *AlertDatasourceService) PrometheusActiveAlerts(ctx context.Context, id uint) (json.RawMessage, error) {
	cli, _, err := s.clientFor(ctx, id)
	if err != nil {
		return nil, err
	}
	qctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	body, _, err := cli.ActiveAlerts(qctx)
	return body, err
}
