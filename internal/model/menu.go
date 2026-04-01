package model

import (
	"time"

	"gorm.io/gorm"
)

type Menu struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ParentID  *uint          `json:"parent_id" gorm:"index"`
	Path      string         `json:"path" gorm:"size:128"`
	Name      string         `json:"name" gorm:"size:64;not null"`
	Icon      string         `json:"icon" gorm:"size:64"`
	Sort      int            `json:"sort" gorm:"default:0"`
	Hidden    bool           `json:"hidden" gorm:"default:false"`
	Component string         `json:"component" gorm:"size:128"`
	Redirect  string         `json:"redirect" gorm:"size:128"`
	Status    int            `json:"status" gorm:"not null;default:1"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Children  []Menu         `json:"children,omitempty" gorm:"-"`
}

func (Menu) TableName() string {
	return "menus"
}
