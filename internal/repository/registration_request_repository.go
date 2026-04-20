package repository

import (
	"context"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/pagination"

	"gorm.io/gorm"
)

// RegistrationRequestRepository 用户自助注册申请的持久化访问。
type RegistrationRequestRepository struct {
	db *gorm.DB
}

// RegistrationRequestListParams 注册申请列表查询条件。
type RegistrationRequestListParams struct {
	Keyword  string
	Status   *int
	Page     int
	PageSize int
}

func NewRegistrationRequestRepository(db *gorm.DB) *RegistrationRequestRepository {
	return &RegistrationRequestRepository{db: db}
}

func (r *RegistrationRequestRepository) Create(ctx context.Context, req *model.RegistrationRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *RegistrationRequestRepository) GetByID(ctx context.Context, id uint) (*model.RegistrationRequest, error) {
	var req model.RegistrationRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *RegistrationRequestRepository) List(ctx context.Context, params RegistrationRequestListParams) ([]model.RegistrationRequest, int64, error) {
	var list []model.RegistrationRequest
	query := r.db.WithContext(ctx).Model(&model.RegistrationRequest{})

	if params.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?",
			"%"+params.Keyword+"%", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	page, pageSize := pagination.Normalize(params.Page, params.PageSize)
	total, err := listWithPagination(query, page, pageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *RegistrationRequestRepository) UpdateStatus(ctx context.Context, id uint, status model.RegistrationRequestStatus, reviewerID uint, comment string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.RegistrationRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         status,
		"reviewer_id":    reviewerID,
		"review_comment": comment,
		"reviewed_at":    now,
	}).Error
}

func (r *RegistrationRequestRepository) CountPending(ctx context.Context, username, email string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RegistrationRequest{}).
		Where("(username = ? OR email = ?) AND status = ?", username, email, model.RegistrationPending).
		Count(&count).Error
	return count, err
}
