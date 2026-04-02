package service

import (
	"context"
	"io"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/pagination"
	"go-permission-system/internal/repository"

	"github.com/xuri/excelize/v2"
)

type OperationLogListQuery struct {
	Method     string `form:"method"`
	Path       string `form:"path"`
	StatusCode *int   `form:"status_code"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type OperationLogService struct {
	repo *repository.OperationLogRepository
}

func NewOperationLogService(repo *repository.OperationLogRepository) *OperationLogService {
	return &OperationLogService{repo: repo}
}

func (s *OperationLogService) Record(ctx context.Context, entry model.OperationLog) error {
	return s.repo.Create(ctx, &entry)
}

func (s *OperationLogService) List(ctx context.Context, query OperationLogListQuery) (*pagination.Result[model.OperationLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, repository.OperationLogListParams{
		Method:     query.Method,
		Path:       query.Path,
		StatusCode: query.StatusCode,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.OperationLog]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *OperationLogService) Delete(ctx context.Context, id uint) error {
	return s.repo.DeleteByID(ctx, id)
}

func (s *OperationLogService) DeleteBatch(ctx context.Context, ids []uint) error {
	return s.repo.DeleteByIDs(ctx, ids)
}

// Export writes operation logs matching query to writer as Excel.
func (s *OperationLogService) Export(ctx context.Context, query OperationLogListQuery, w io.Writer) error {
	page := 1
	pageSize := 1000000
	list, _, err := s.repo.List(ctx, repository.OperationLogListParams{
		Method:     query.Method,
		Path:       query.Path,
		StatusCode: query.StatusCode,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return err
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]interface{}{"ID", "Method", "Path", "StatusCode", "LatencyMs", "IP", "RequestHeaders", "RequestBody", "ResponseBody", "CreatedAt", "User"})
	for i, l := range list {
		row := []interface{}{
			l.ID,
			l.Method,
			l.Path,
			l.StatusCode,
			l.LatencyMs,
			l.IP,
			l.RequestHeaders,
			l.RequestBody,
			l.ResponseBody,
			l.CreatedAt,
			l.Username,
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	return f.Write(w)
}
