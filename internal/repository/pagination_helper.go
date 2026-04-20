package repository

import (
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

func listWithPagination[T any](query *gorm.DB, page, pageSize int, order string, out *[]T) (int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	page, pageSize = pagination.Normalize(page, pageSize)
	offset := (page - 1) * pageSize
	if err := query.Order(order).Offset(offset).Limit(pageSize).Find(out).Error; err != nil {
		return 0, err
	}
	return total, nil
}
