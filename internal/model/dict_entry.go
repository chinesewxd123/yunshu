package model

import (
	"time"

	"gorm.io/gorm"
)

// DictEntry 数据字典条目。
// value 使用 MEDIUMTEXT 且不参与索引，避免 VARCHAR(4096) 无法容纳完整 kubeconfig（含证书 base64）。
type DictEntry struct {
	ID        uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	DictType  string         `json:"dict_type" gorm:"size:64;not null;index:idx_dict_entry_type;comment:字典类型"`
	Label     string         `json:"label" gorm:"size:128;not null;comment:显示标签"`
	Value     string         `json:"value" gorm:"type:mediumtext;not null;comment:字典值（如完整 kubeconfig）"`
	Sort      int            `json:"sort" gorm:"not null;default:0;comment:排序"`
	Status    int            `json:"status" gorm:"not null;default:1;comment:状态 1启用 0停用"`
	Remark    string         `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名。
func (DictEntry) TableName() string {
	return "dict_entries"
}
