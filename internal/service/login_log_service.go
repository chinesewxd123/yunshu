package service

import (
	"context"
	"io"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/repository"

	"github.com/xuri/excelize/v2"
)

type LoginLogListQuery struct {
	Username string `form:"username"`
	Status   *int   `form:"status"` // 1 成功 0 失败
	Source   string `form:"source"` // password | email
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type LoginLogService struct {
	repo *repository.LoginLogRepository
}

func NewLoginLogService(repo *repository.LoginLogRepository) *LoginLogService {
	return &LoginLogService{repo: repo}
}

func (s *LoginLogService) Record(ctx context.Context, entry model.LoginLog) error {
	return s.repo.Create(ctx, &entry)
}

func (s *LoginLogService) List(ctx context.Context, query LoginLogListQuery) (*pagination.Result[model.LoginLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, repository.LoginLogListParams{
		Username: query.Username,
		Status:   query.Status,
		Source:   query.Source,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.LoginLog]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *LoginLogService) Delete(ctx context.Context, id uint) error {
	return s.repo.DeleteByID(ctx, id)
}

func (s *LoginLogService) DeleteBatch(ctx context.Context, ids []uint) error {
	return s.repo.DeleteByIDs(ctx, ids)
}

// Export writes login logs matching query to writer as Excel.
func (s *LoginLogService) Export(ctx context.Context, query LoginLogListQuery, w io.Writer) error {
	// fetch a large page to include all records matching filters
	page := 1
	pageSize := 1000000
	list, _, err := s.repo.List(ctx, repository.LoginLogListParams{
		Username: query.Username,
		Status:   query.Status,
		Source:   query.Source,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return err
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]interface{}{"ID", "Username", "IP", "Source", "Status", "Detail", "UserAgent", "CreatedAt"})
	for i, l := range list {
		row := []interface{}{l.ID, l.Username, l.IP, l.Source, l.Status, l.Detail, l.UserAgent, l.CreatedAt}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	return f.Write(w)
}
