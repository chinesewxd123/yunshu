package model

import (
	"time"

	"gorm.io/gorm"
)

// ServerCredential SSH 登录凭据：密码或密钥材料均以密文存储，不落明文。
type ServerCredential struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ServerID uint   `json:"server_id" gorm:"not null;uniqueIndex;comment:服务器ID"`
	AuthType string `json:"auth_type" gorm:"size:16;not null;default:'password';comment:认证类型"` // password/key
	Username string `json:"username" gorm:"size:128;not null;comment:登录用户名"`

	// 从数据字典选择模板时记录标签，便于编辑弹窗回显（非密钥本身）。
	UsernameDictLabel *string `json:"username_dict_label,omitempty" gorm:"size:191;comment:字典用户名模板标签"`
	PasswordDictLabel *string `json:"password_dict_label,omitempty" gorm:"size:191;comment:字典密码模板标签"`

	EncPassword    *string `json:"-" gorm:"type:longtext;comment:加密后的密码"`
	EncPrivateKey  *string `json:"-" gorm:"type:longtext;comment:加密后的私钥"`
	EncPassphrase  *string `json:"-" gorm:"type:longtext;comment:加密后的私钥口令"`
	KeyVersion     int     `json:"key_version" gorm:"not null;default:1;comment:密钥版本号"`
	FingerprintSHA string  `json:"fingerprint_sha" gorm:"size:64;index;comment:凭据指纹SHA"` // optional: for audit / rotation

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 server_credentials。
func (ServerCredential) TableName() string { return "server_credentials" }
