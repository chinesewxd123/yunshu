package service

import (
	"context"
	"errors"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type K8sEventForwardAdminService struct {
	db *gorm.DB
}

func NewK8sEventForwardAdminService(db *gorm.DB) *K8sEventForwardAdminService {
	return &K8sEventForwardAdminService{db: db}
}

type K8sEventForwardRuleListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (s *K8sEventForwardAdminService) ListRules(ctx context.Context, q K8sEventForwardRuleListQuery) (*pagination.Result[model.K8sEventForwardRule], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	var total int64
	if err := s.db.WithContext(ctx).Model(&model.K8sEventForwardRule{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.K8sEventForwardRule
	err := s.db.WithContext(ctx).Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.K8sEventForwardRule]{
		List: list, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (s *K8sEventForwardAdminService) GetRule(ctx context.Context, id uint) (*model.K8sEventForwardRule, error) {
	var rule model.K8sEventForwardRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	return &rule, nil
}

func (s *K8sEventForwardAdminService) CreateRule(ctx context.Context, rule *model.K8sEventForwardRule) error {
	return s.db.WithContext(ctx).Create(rule).Error
}

func (s *K8sEventForwardAdminService) UpdateRule(ctx context.Context, rule *model.K8sEventForwardRule) error {
	return s.db.WithContext(ctx).Save(rule).Error
}

func (s *K8sEventForwardAdminService) DeleteRule(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.K8sEventForwardRule{}, id).Error
}

func (s *K8sEventForwardAdminService) GetSettings(ctx context.Context) (*model.K8sEventForwardSetting, error) {
	var st model.K8sEventForwardSetting
	err := s.db.WithContext(ctx).First(&st, 1).Error
	if err == gorm.ErrRecordNotFound {
		st.ID = 1
		return &st, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *K8sEventForwardAdminService) UpdateSettings(ctx context.Context, st *model.K8sEventForwardSetting) error {
	st.ID = 1
	return s.db.WithContext(ctx).Save(st).Error
}
