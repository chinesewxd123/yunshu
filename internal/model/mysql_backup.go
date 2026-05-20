package model

import (
	"time"

	"gorm.io/gorm"
)

// MysqlBackupMode 备份模式：mysqldump 逻辑备份；xtrabackup 物理备份（经 SSH 执行 xtrabackup）。
const (
	MysqlBackupModeMysqldump   = "mysqldump"
	MysqlBackupModeXtrabackup  = "xtrabackup"
	MysqlBackupModeRemoteCheck = "remote_check" // 兼容旧数据，等同 xtrabackup
	// MysqlBackupExecXtrabackup 任务记录的实际执行方式
	MysqlBackupExecXtrabackup = "xtrabackup"
)

// MysqlBackupScope mysqldump 备份范围。
const (
	MysqlBackupScopeAll      = "all"
	MysqlBackupScopeDatabase = "database"
	MysqlBackupScopeTable    = "table"
)

// MysqlBackupTrigger 任务触发方式。
const (
	MysqlBackupTriggerManual    = "manual"
	MysqlBackupTriggerScheduled = "scheduled"
)

// MysqlBackupInstance 项目内 MySQL 备份目标（复用服务器 SSH 凭据）。
type MysqlBackupInstance struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;index:idx_mysql_bak_proj"`
	ServerID  uint   `json:"server_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"size:128;not null"`
	Enabled   bool   `json:"enabled" gorm:"default:true;index"`

	MysqlHost     string `json:"mysql_host" gorm:"size:255;not null;default:'127.0.0.1'"`
	MysqlPort     int    `json:"mysql_port" gorm:"not null;default:3306"`
	MysqlUser     string `json:"mysql_user" gorm:"size:128;not null"`
	EncPassword   string `json:"-" gorm:"type:longtext;comment:加密后的 MySQL 密码"`
	BackupMode    string `json:"backup_mode" gorm:"size:32;not null;default:'mysqldump'"`

	// BackupScope：all | database | table（mysqldump 模式）
	BackupScope string `json:"backup_scope" gorm:"size:32;not null;default:'all'"`
	// DatabaseName / BackupTable 单库、单表备份
	DatabaseName string `json:"database_name" gorm:"size:128"`
	BackupTable  string `json:"table_name" gorm:"column:table_name;size:128"`
	// DatabaseNames 历史字段：逗号分隔多库（未指定 scope 时兼容）
	DatabaseNames string `json:"database_names" gorm:"type:text"`

	RemoteDataDir string `json:"remote_data_dir" gorm:"size:512"`
	RemoteLogDir  string `json:"remote_log_dir" gorm:"size:512"`
	// MysqlDataDir 宿主机上 MySQL 真实 datadir（Docker 部署时填挂载目录，如 /export/mysql_data，非容器内 /var/lib/mysql）
	MysqlDataDir string `json:"mysql_datadir" gorm:"size:512"`
	// UploadToMinio 为 true 时将备份产物上传 MinIO；为 false 时仅执行备份/检查并保留远端文件。
	UploadToMinio bool `json:"upload_to_minio" gorm:"not null;default:true"`
	// MysqldumpWorkDir 远端 mysqldump 落盘目录（绝对路径，非 /tmp）
	MysqldumpWorkDir string `json:"mysqldump_work_dir" gorm:"size:512"`
	// MysqldumpOptions JSON 字符串数组，选项 id 见 mysqlbackup.MysqldumpOptionCatalog
	MysqldumpOptions string `json:"mysqldump_options" gorm:"type:text"`
	// MysqldumpExtraArgs 额外 mysqldump 参数（空格分隔，须以 - 开头）
	MysqldumpExtraArgs string `json:"mysqldump_extra_args" gorm:"size:512"`

	ScheduleEnabled bool       `json:"schedule_enabled" gorm:"not null;default:false;index"`
	CronSpec        string     `json:"cron_spec" gorm:"size:256;not null;default:''"`
	LastScheduledAt *time.Time `json:"last_scheduled_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (MysqlBackupInstance) TableName() string { return "mysql_backup_instances" }

// MysqlBackupJob 单次备份任务记录。
type MysqlBackupJob struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	InstanceID uint   `json:"instance_id" gorm:"not null;index"`
	ProjectID  uint   `json:"project_id" gorm:"not null;index"`
	Status     string `json:"status" gorm:"size:32;not null;index"` // pending/running/success/failed
	BackupMode string `json:"backup_mode" gorm:"size:32"`
	TriggerType string `json:"trigger_type" gorm:"size:32;not null;default:'manual'"`
	BackupScope string `json:"backup_scope,omitempty" gorm:"size:32"`
	DatabaseName string `json:"database_name,omitempty" gorm:"size:128"`
	BackupTable  string `json:"table_name,omitempty" gorm:"column:table_name;size:128"`

	RemotePath   string `json:"remote_path,omitempty" gorm:"size:512"`
	MinioBucket  string `json:"minio_bucket,omitempty" gorm:"size:128"`
	MinioObject  string `json:"minio_object,omitempty" gorm:"size:512"`
	FileSize     int64  `json:"file_size" gorm:"default:0"`
	CheckOK      bool   `json:"check_ok" gorm:"default:false"`
	LogExcerpt   string `json:"log_excerpt,omitempty" gorm:"type:text"`
	ErrorMessage string `json:"error_message,omitempty" gorm:"type:text"`

	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (MysqlBackupJob) TableName() string { return "mysql_backup_jobs" }
