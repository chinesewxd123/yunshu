package repository

import (
	"context"
	"strings"

	"go-permission-system/internal/model"

	"gorm.io/gorm"
)

type DictEntryRepository struct {
	db *gorm.DB
}

func NewDictEntryRepository(db *gorm.DB) *DictEntryRepository {
	return &DictEntryRepository{db: db}
}

func (r *DictEntryRepository) Create(ctx context.Context, item *model.DictEntry) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *DictEntryRepository) Update(ctx context.Context, item *model.DictEntry) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *DictEntryRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DictEntry{}, id).Error
}

func (r *DictEntryRepository) DeleteByTypeAndLabel(ctx context.Context, dictType, label string) error {
	return r.db.WithContext(ctx).
		Where("dict_type = ? AND label = ?", strings.TrimSpace(dictType), strings.TrimSpace(label)).
		Delete(&model.DictEntry{}).Error
}

func (r *DictEntryRepository) DeleteByTypes(ctx context.Context, dictTypes []string) error {
	if len(dictTypes) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(dictTypes))
	for _, item := range dictTypes {
		if one := strings.TrimSpace(item); one != "" {
			trimmed = append(trimmed, one)
		}
	}
	if len(trimmed) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("dict_type IN ?", trimmed).
		Delete(&model.DictEntry{}).Error
}

func (r *DictEntryRepository) GetByID(ctx context.Context, id uint) (*model.DictEntry, error) {
	var item model.DictEntry
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DictEntryRepository) ExistsByTypeValue(ctx context.Context, dictType, value string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND value = ?", strings.TrimSpace(dictType), strings.TrimSpace(value))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DictEntryRepository) ExistsByType(ctx context.Context, dictType string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ?", strings.TrimSpace(dictType))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DictEntryRepository) ExistsByTypeLabel(ctx context.Context, dictType, label string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND label = ?", strings.TrimSpace(dictType), strings.TrimSpace(label))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DictEntryRepository) DeleteByTypeAndValue(ctx context.Context, dictType, value string) error {
	return r.db.WithContext(ctx).
		Where("dict_type = ? AND value = ?", strings.TrimSpace(dictType), strings.TrimSpace(value)).
		Delete(&model.DictEntry{}).Error
}

// CleanupDuplicateTypeValue 删除重复字典项（按 dict_type + TRIM(value) 维度，仅保留最小 id）。
func (r *DictEntryRepository) CleanupDuplicateTypeValue(ctx context.Context) error {
	sql := `
DELETE d1
FROM dict_entries d1
JOIN dict_entries d2
  ON d1.dict_type = d2.dict_type
 AND TRIM(d1.value) = TRIM(d2.value)
 AND d1.id > d2.id
WHERE d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL
`
	return r.db.WithContext(ctx).Exec(sql).Error
}

// CleanupDuplicateTypeLabel 删除重复字典项（按 dict_type + TRIM(label) 维度，仅保留最小 id）。
func (r *DictEntryRepository) CleanupDuplicateTypeLabel(ctx context.Context) error {
	sql := `
DELETE d1
FROM dict_entries d1
JOIN dict_entries d2
  ON d1.dict_type = d2.dict_type
 AND TRIM(d1.label) = TRIM(d2.label)
 AND d1.id > d2.id
WHERE d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL
`
	return r.db.WithContext(ctx).Exec(sql).Error
}

func (r *DictEntryRepository) List(ctx context.Context, dictType, keyword string, status *int, page, pageSize int) ([]model.DictEntry, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.DictEntry{})
	if strings.TrimSpace(dictType) != "" {
		query = query.Where("dict_type = ?", strings.TrimSpace(dictType))
	}
	if strings.TrimSpace(keyword) != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("label LIKE ? OR value LIKE ? OR remark LIKE ?", like, like, like)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	var list []model.DictEntry
	err := query.Order("id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *DictEntryRepository) ListByTypeEnabled(ctx context.Context, dictType string) ([]model.DictEntry, error) {
	var list []model.DictEntry
	err := r.db.WithContext(ctx).
		Where("dict_type = ? AND status = 1", strings.TrimSpace(dictType)).
		Order("sort ASC, id ASC").
		Find(&list).Error
	return list, err
}
