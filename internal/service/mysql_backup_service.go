package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/mysqlbackup"
	"yunshu/internal/service/svclog"
	"yunshu/internal/pkg/objectstore"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/repository"
	"yunshu/internal/service/svcerr"

	"crypto/cipher"
	"gorm.io/gorm"
)

type MysqlBackupService struct {
	backupRepo *repository.MysqlBackupRepository
	serverRepo *repository.ServerRepository
	projectRepo *repository.ProjectRepository
	db          *gorm.DB
	aead        cipher.AEAD
	schedMu     sync.Mutex
	schedRunning map[uint]bool
	bizLog       *logx.Component
}

func NewMysqlBackupService(
	backupRepo *repository.MysqlBackupRepository,
	serverRepo *repository.ServerRepository,
	projectRepo *repository.ProjectRepository,
	db *gorm.DB,
	encryptionKey string,
) (*MysqlBackupService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	return &MysqlBackupService{
		backupRepo:   backupRepo,
		serverRepo:   serverRepo,
		projectRepo:  projectRepo,
		db:           db,
		aead:         aead,
		schedRunning: make(map[uint]bool),
		bizLog:       svclog.Worker("mysql.backup"),
	}, nil
}

func (s *MysqlBackupService) SetBizLog(c *logx.Component) {
	s.bizLog = c
}

type MysqlBackupInstanceItem struct {
	ID            uint   `json:"id"`
	ProjectID     uint   `json:"project_id"`
	ServerID      uint   `json:"server_id"`
	ServerName    string `json:"server_name,omitempty"`
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	MysqlHost     string `json:"mysql_host"`
	MysqlPort     int    `json:"mysql_port"`
	MysqlUser     string `json:"mysql_user"`
	BackupMode    string `json:"backup_mode"`
	BackupScope   string `json:"backup_scope"`
	DatabaseName  string `json:"database_name"`
	TableName     string `json:"table_name"`
	DatabaseNames string `json:"database_names"`
	RemoteDataDir      string   `json:"remote_data_dir"`
	RemoteLogDir       string   `json:"remote_log_dir"`
	MysqlDataDir       string   `json:"mysql_datadir"`
	UploadToMinio      bool     `json:"upload_to_minio"`
	MysqldumpWorkDir   string   `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string `json:"mysqldump_options"`
	MysqldumpExtraArgs string   `json:"mysqldump_extra_args"`
	ScheduleEnabled    bool     `json:"schedule_enabled"`
	CronSpec        string `json:"cron_spec"`
	LastScheduledAt string `json:"last_scheduled_at,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type MysqlBackupInstanceUpsertRequest struct {
	ProjectID     uint   `json:"project_id"`
	ServerID      uint   `json:"server_id" binding:"required"`
	Name          string `json:"name" binding:"required,max=128"`
	Enabled       *bool  `json:"enabled"`
	MysqlHost     string `json:"mysql_host"`
	MysqlPort     int    `json:"mysql_port"`
	MysqlUser     string `json:"mysql_user" binding:"required"`
	MysqlPassword string `json:"mysql_password"`
	BackupMode    string `json:"backup_mode"`
	BackupScope   string `json:"backup_scope"`
	DatabaseName  string `json:"database_name"`
	TableName     string `json:"table_name"`
	DatabaseNames string `json:"database_names"`
	RemoteDataDir      string   `json:"remote_data_dir"`
	RemoteLogDir       string   `json:"remote_log_dir"`
	MysqlDataDir       string   `json:"mysql_datadir"`
	UploadToMinio      *bool    `json:"upload_to_minio"`
	MysqldumpWorkDir   string   `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string `json:"mysqldump_options"`
	MysqldumpExtraArgs string   `json:"mysqldump_extra_args"`
	ScheduleEnabled    *bool    `json:"schedule_enabled"`
	CronSpec        string `json:"cron_spec"`
}

type MysqlBackupInstanceListQuery struct {
	ProjectID uint `form:"project_id"`
	Page      int  `form:"page"`
	PageSize  int  `form:"page_size"`
}

type MysqlBackupJobListQuery struct {
	ProjectID  uint `form:"project_id"`
	InstanceID uint `form:"instance_id"`
	Page       int  `form:"page"`
	PageSize   int  `form:"page_size"`
}

func (s *MysqlBackupService) ListInstances(ctx context.Context, q MysqlBackupInstanceListQuery) (*pagination.Result[MysqlBackupInstanceItem], error) {
	list, total, err := s.backupRepo.ListInstances(ctx, repository.MysqlBackupInstanceListParams{
		ProjectID: q.ProjectID, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, svcerr.Pass(ctx, "mysql.backup", "ListInstances", err)
	}
	out := make([]MysqlBackupInstanceItem, 0, len(list))
	for _, inst := range list {
		out = append(out, s.toInstanceItem(ctx, inst))
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	return &pagination.Result[MysqlBackupInstanceItem]{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *MysqlBackupService) UpsertInstance(ctx context.Context, id uint, req MysqlBackupInstanceUpsertRequest) (*MysqlBackupInstanceItem, error) {
	if err := s.ensureServerInProject(ctx, req.ProjectID, req.ServerID); err != nil {
		return nil, err
	}
	mode := strings.TrimSpace(req.BackupMode)
	if mode == "" {
		mode = model.MysqlBackupModeMysqldump
	}
	if mode != model.MysqlBackupModeMysqldump && !model.IsXtrabackupBackupMode(mode) {
		return nil, constants.ErrBadRequestWithMsg("backup_mode 须为 mysqldump 或 xtrabackup")
	}
	if model.IsXtrabackupBackupMode(mode) {
		mode = model.MysqlBackupModeXtrabackup
	}

	var inst *model.MysqlBackupInstance
	if id > 0 {
		existing, err := s.backupRepo.GetInstanceInProject(ctx, req.ProjectID, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFound
			}
			return nil, svcerr.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
		inst = existing
	} else {
		inst = &model.MysqlBackupInstance{ProjectID: req.ProjectID}
	}

	inst.ServerID = req.ServerID
	inst.Name = strings.TrimSpace(req.Name)
	if req.Enabled != nil {
		inst.Enabled = *req.Enabled
	} else if id == 0 {
		inst.Enabled = true
	}
	inst.MysqlHost = strings.TrimSpace(req.MysqlHost)
	if inst.MysqlHost == "" {
		inst.MysqlHost = "127.0.0.1"
	}
	if req.MysqlPort > 0 {
		inst.MysqlPort = req.MysqlPort
	} else if inst.MysqlPort <= 0 {
		inst.MysqlPort = 3306
	}
	inst.MysqlUser = strings.TrimSpace(req.MysqlUser)
	inst.BackupMode = mode
	scope := strings.TrimSpace(req.BackupScope)
	if scope == "" {
		scope = model.MysqlBackupScopeAll
	}
	if mode == model.MysqlBackupModeMysqldump {
		if err := validateMysqlBackupScope(scope, req.DatabaseName, req.TableName, req.DatabaseNames); err != nil {
			return nil, err
		}
		inst.BackupScope = scope
		inst.DatabaseName = strings.TrimSpace(req.DatabaseName)
		inst.BackupTable = strings.TrimSpace(req.TableName)
	} else {
		inst.BackupScope = model.MysqlBackupScopeAll
		if strings.TrimSpace(req.RemoteDataDir) == "" || strings.TrimSpace(req.RemoteLogDir) == "" {
			return nil, constants.ErrBadRequestWithMsg("xtrabackup 模式须填写 remote_data_dir 与 remote_log_dir")
		}
		if strings.TrimSpace(req.MysqlDataDir) == "" {
			return nil, constants.ErrBadRequestWithMsg("xtrabackup 模式须填写 mysql_datadir（宿主机 MySQL 数据目录，Docker 常为 /export/mysql_data）")
		}
	}
	inst.DatabaseNames = strings.TrimSpace(req.DatabaseNames)
	inst.RemoteDataDir = strings.TrimSpace(req.RemoteDataDir)
	inst.RemoteLogDir = strings.TrimSpace(req.RemoteLogDir)
	inst.MysqlDataDir = strings.TrimSpace(req.MysqlDataDir)
	if req.UploadToMinio != nil {
		inst.UploadToMinio = *req.UploadToMinio
	} else if id == 0 {
		inst.UploadToMinio = true
	}

	workDir, err := mysqlbackup.NormalizeMysqldumpWorkDir(req.MysqldumpWorkDir)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqldumpWorkDir = workDir
	if err := mysqlbackup.ValidateBackupPathIsolation(workDir, inst.RemoteDataDir, inst.RemoteLogDir); err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	optionsJSON := marshalMysqldumpOptionIDs(req.MysqldumpOptions)
	optIDs, err := mysqlbackup.ParseMysqldumpOptionIDs(optionsJSON)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	if _, err := mysqlbackup.FormatMysqldumpFlags(optIDs, req.MysqldumpExtraArgs); err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	inst.MysqldumpOptions = optionsJSON
	inst.MysqldumpExtraArgs = strings.TrimSpace(req.MysqldumpExtraArgs)

	if req.ScheduleEnabled != nil {
		inst.ScheduleEnabled = *req.ScheduleEnabled
	}
	cronSpec := strings.TrimSpace(req.CronSpec)
	if cronSpec != "" || (req.ScheduleEnabled != nil && *req.ScheduleEnabled) {
		if err := ValidateMysqlBackupCronSpec(cronSpec); err != nil {
			return nil, err
		}
		inst.CronSpec = cronSpec
	} else if id == 0 {
		inst.CronSpec = ""
	}
	if inst.ScheduleEnabled && strings.TrimSpace(inst.CronSpec) == "" {
		return nil, constants.ErrBadRequestWithMsg("启用定时备份时必须填写 cron_spec（Cron 表达式）")
	}

	if pw := strings.TrimSpace(req.MysqlPassword); pw != "" {
		enc, err := cryptox.EncryptString(s.aead, pw)
		if err != nil {
			return nil, svcerr.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
		inst.EncPassword = enc
	} else if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("新建实例须填写 mysql_password")
	}

	if id > 0 {
		if err := s.backupRepo.UpdateInstance(ctx, inst); err != nil {
			return nil, svcerr.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
	} else {
		if err := s.backupRepo.CreateInstance(ctx, inst); err != nil {
			return nil, svcerr.Pass(ctx, "mysql.backup", "UpsertInstance", err)
		}
	}
	item := s.toInstanceItem(ctx, *inst)
	return &item, nil
}

func (s *MysqlBackupService) DeleteInstance(ctx context.Context, projectID, id uint) error {
	if _, err := s.backupRepo.GetInstanceInProject(ctx, projectID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return svcerr.Pass(ctx, "mysql.backup", "DeleteInstance", err)
	}
	return s.backupRepo.DeleteInstance(ctx, id)
}

func (s *MysqlBackupService) PingInstance(ctx context.Context, projectID, instanceID uint) (bool, string, error) {
	inst, pw, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return false, "", err
	}
	if err := mysqlbackup.Ping(ctx, inst.MysqlHost, inst.MysqlPort, inst.MysqlUser, pw); err != nil {
		return false, fmt.Sprintf("mysqlping,host=%s,port=%d status=0i", inst.MysqlHost, inst.MysqlPort), nil
	}
	return true, fmt.Sprintf("mysqlping,host=%s,port=%d status=1i", inst.MysqlHost, inst.MysqlPort), nil
}

func (s *MysqlBackupService) findLatestBackupArtifact(ctx context.Context, inst *model.MysqlBackupInstance) (*mysqlbackup.BackupArtifact, error) {
	prefix, err := s.backupArtifactNamePrefix(ctx, inst)
	if err != nil {
		return nil, err
	}
	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return nil, err
	}
	defer sshCli.Close()
	script := mysqlbackup.BuildFindLatestBackupScript(inst.RemoteDataDir, inst.RemoteLogDir, prefix, 30)
	res, err := sshCli.Exec(ctx, script, 16384)
	if err != nil && !strings.Contains(res.Stdout+res.Stderr, "NOT_FOUND") {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHExecFailedPrefix + err.Error())
	}
	artifact := mysqlbackup.ParseFindLatestBackupOutput(strings.TrimSpace(res.Stdout+"\n"+res.Stderr), inst.MysqlPort)
	if artifact.OK {
		artifact.Message = fmt.Sprintf("找到有效备份: %s", artifact.BackupFile)
	} else {
		artifact.Message = fmt.Sprintf("未找到匹配前缀 %q 的有效备份（*.tar.gz 且日志末行含 %s）", prefix, mysqlbackup.BackupCompletedMarker)
	}
	return &artifact, nil
}

func (s *MysqlBackupService) CheckRemoteBackup(ctx context.Context, projectID, instanceID uint, dayOffset int) (*mysqlbackup.RemoteCheckResult, error) {
	inst, _, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if !model.IsXtrabackupBackupMode(inst.BackupMode) {
		return nil, constants.ErrBadRequestWithMsg("该实例不是 xtrabackup 模式")
	}
	_ = dayOffset
	artifact, err := s.findLatestBackupArtifact(ctx, inst)
	if err != nil {
		return nil, err
	}
	return &mysqlbackup.RemoteCheckResult{
		BackupFile:   artifact.BackupFile,
		LogFile:      artifact.LogFile,
		LogCompleted: artifact.OK,
		OK:           artifact.OK,
		Message:      artifact.Message,
	}, nil
}

func (s *MysqlBackupService) RunBackup(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupJob, error) {
	return s.enqueueBackup(ctx, projectID, instanceID, model.MysqlBackupTriggerManual)
}

func (s *MysqlBackupService) enqueueBackup(ctx context.Context, projectID, instanceID uint, trigger string) (*model.MysqlBackupJob, error) {
	n, _ := s.backupRepo.FailStaleRunningJobs(ctx, 2*time.Hour)
	if n > 0 && s.bizLog != nil {
		s.bizLog.Warnw("Marked stale MySQL backup jobs as failed", "count", n)
	}
	inst, _, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if !inst.Enabled {
		return nil, constants.ErrBadRequestWithMsg("备份实例已停用")
	}
	running, err := s.backupRepo.HasRunningJob(ctx, instanceID)
	if err != nil {
		return nil, svcerr.Pass(ctx, "mysql.backup", "enqueueBackup", err)
	}
	if running {
		return nil, constants.ErrBadRequestWithMsg("该实例已有进行中的备份任务")
	}

	target := mysqlbackup.BuildDumpTarget(inst)
	now := time.Now()
	job := &model.MysqlBackupJob{
		InstanceID:   inst.ID,
		ProjectID:    projectID,
		Status:       "running",
		BackupMode:   inst.BackupMode,
		TriggerType:  trigger,
		BackupScope:  target.Scope,
		DatabaseName: target.Database,
		BackupTable:  target.Table,
		StartedAt:    &now,
	}
	if err := s.backupRepo.CreateJob(ctx, job); err != nil {
		return nil, svcerr.Pass(ctx, "mysql.backup", "enqueueBackup", err)
	}

	go s.runBackupJobAsync(job.ID, projectID, instanceID, trigger)
	return job, nil
}

const mysqlBackupJobTimeout = 35 * time.Minute

const mysqlXtrabackupJobTimeout = 2 * time.Hour

func (s *MysqlBackupService) runBackupJobAsync(jobID, projectID, instanceID uint, trigger string) {
	timeout := mysqlBackupJobTimeout
	if inst, err := s.backupRepo.GetInstanceInProject(context.Background(), projectID, instanceID); err == nil && model.IsXtrabackupBackupMode(inst.BackupMode) {
		timeout = mysqlXtrabackupJobTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	s.finishBackupJob(ctx, jobID, projectID, instanceID, trigger)
}

func (s *MysqlBackupService) finishBackupJob(ctx context.Context, jobID, projectID, instanceID uint, trigger string) {
	started := time.Now()
	job, err := s.backupRepo.GetJob(ctx, jobID)
	if err != nil {
		return
	}
	inst, pw, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		fin := time.Now()
		job.FinishedAt = &fin
		_ = s.backupRepo.UpdateJob(ctx, job)
		s.logBackupJobDone(jobID, 0, "", trigger, "failed", time.Since(started), err)
		return
	}
	s.logBackupJobBegin(jobID, inst, trigger)
	target := mysqlbackup.BuildDumpTarget(inst)

	var runErr error
	switch {
	case model.IsXtrabackupBackupMode(inst.BackupMode):
		runErr = s.runXtrabackupUpload(ctx, inst, pw, job)
	default:
		runErr = s.runMysqldumpUpload(ctx, inst, pw, job, target, "")
	}

	fin := time.Now()
	job.FinishedAt = &fin
	if runErr != nil {
		job.Status = "failed"
		job.ErrorMessage = runErr.Error()
	} else {
		job.Status = "success"
	}
	_ = s.backupRepo.UpdateJob(ctx, job)
	s.logBackupJobDone(jobID, inst.ID, inst.Name, trigger, job.Status, time.Since(started), runErr,
		"minio_object", job.MinioObject,
		"file_size", job.FileSize,
		"remote_path", job.RemotePath,
	)
}

func (s *MysqlBackupService) logBackupJobBegin(jobID uint, inst *model.MysqlBackupInstance, trigger string) {
	if s.bizLog == nil || inst == nil {
		return
	}
	s.bizLog.Infow("Started MySQL backup job",
		"job_id", jobID,
		"instance_id", inst.ID,
		"project_id", inst.ProjectID,
		"instance_name", inst.Name,
		"backup_mode", inst.BackupMode,
		"trigger", trigger,
		"mysql_user", inst.MysqlUser,
		"mysql_host", inst.MysqlHost,
		"mysql_port", inst.MysqlPort,
	)
}

func (s *MysqlBackupService) logBackupJobDone(jobID, instanceID uint, instanceName, trigger, status string, dur time.Duration, runErr error, extra ...any) {
	if s.bizLog == nil {
		return
	}
	attrs := []any{
		"job_id", jobID,
		"instance_id", instanceID,
		"instance_name", instanceName,
		"trigger", trigger,
		"status", status,
		"duration_ms", dur.Milliseconds(),
	}
	attrs = append(attrs, extra...)
	if runErr != nil {
		s.bizLog.Errorw(runErr, "Failed to finish MySQL backup job", attrs...)
		return
	}
	s.bizLog.Infow("Finished MySQL backup job", attrs...)
}

func (s *MysqlBackupService) logBackupPhase(jobID uint, phase string, attrs ...any) {
	if s.bizLog == nil {
		return
	}
	base := []any{"job_id", jobID, "phase", phase}
	s.bizLog.Infow("MySQL backup job phase", append(base, attrs...)...)
}

func validateMysqlBackupScope(scope, dbName, tableName, databaseNames string) error {
	scope = strings.TrimSpace(scope)
	switch scope {
	case model.MysqlBackupScopeTable:
		if strings.TrimSpace(dbName) == "" || strings.TrimSpace(tableName) == "" {
			return constants.ErrBadRequestWithMsg("单表备份须填写 database_name 与 table_name")
		}
	case model.MysqlBackupScopeDatabase:
		if strings.TrimSpace(dbName) == "" && strings.TrimSpace(databaseNames) == "" {
			return constants.ErrBadRequestWithMsg("单库备份须填写 database_name 或 database_names")
		}
	case model.MysqlBackupScopeAll, "":
		return nil
	default:
		return constants.ErrBadRequestWithMsg("backup_scope 须为 all、database 或 table")
	}
	return nil
}

func (s *MysqlBackupService) runMysqldumpUpload(ctx context.Context, inst *model.MysqlBackupInstance, pw string, job *model.MysqlBackupJob, target mysqlbackup.DumpTarget, logPrefix string) error {
	sshCli, sv, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()
	s.logBackupPhase(job.ID, "ssh_connected", "server_id", inst.ServerID)

	workDir, err := mysqlbackup.NormalizeMysqldumpWorkDir(inst.MysqldumpWorkDir)
	if err != nil {
		return err
	}
	optIDs, err := mysqlbackup.ParseMysqldumpOptionIDs(inst.MysqldumpOptions)
	if err != nil {
		return err
	}
	dumpFlags, err := mysqlbackup.FormatMysqldumpFlags(optIDs, inst.MysqldumpExtraArgs)
	if err != nil {
		return err
	}

	startedAt := time.Now().UTC()
	basename, err := s.backupArtifactBasename(ctx, inst, startedAt)
	if err != nil {
		return err
	}
	remotePath := filepath.ToSlash(filepath.Join(workDir, basename+".sql.gz"))
	logPath := filepath.ToSlash(filepath.Join(workDir, basename+".log"))
	job.RemotePath = remotePath
	job.BackupMode = model.MysqlBackupModeMysqldump

	dumpTarget := mysqlbackup.FormatDumpArgsShell(target, shellQuote)

	dumpCmd := mysqlbackup.BuildMysqldumpRemoteScript(mysqlbackup.MysqldumpRemoteScriptParams{
		WorkDir: workDir, Basename: basename,
		MySQLHost: inst.MysqlHost, MySQLPort: inst.MysqlPort, MySQLUser: inst.MysqlUser,
		MySQLPass: shellQuote(pw), DumpFlags: dumpFlags, DumpTarget: dumpTarget, ShellQuote: shellQuote,
	})
	stopPoll := s.startPollBackupJobLog(ctx, job.ID, sshCli, logPath, logPrefix)
	defer stopPoll()

	s.logBackupPhase(job.ID, "mysqldump_start",
		"work_dir", workDir,
		"basename", basename,
		"remote_sql_gz", remotePath,
		"flags", dumpFlags,
	)
	res, err := sshCli.Exec(ctx, dumpCmd, 65536)
	job.LogExcerpt = mysqlbackup.TruncateLog(logPrefix + strings.TrimSpace(res.Stdout+res.Stderr))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(res.Stderr+res.Stdout))
	}
	s.logBackupPhase(job.ID, "mysqldump_done", "exit_code", res.ExitCode, "ssh_duration_ms", res.Duration.Milliseconds())

	if !inst.UploadToMinio {
		s.logBackupPhase(job.ID, "skip_minio_upload", "remote_sql_gz", remotePath, "remote_log", logPath)
		job.CheckOK = true
		_ = sv
		return nil
	}

	minioCli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "minio_client_ready",
		"endpoint", minioCli.Endpoint(),
		"bucket", minioCli.Bucket(),
	)

	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err := sshCli.WaitRemoteFile(waitCtx, remotePath, 1024, 30*time.Minute); err != nil {
		return err
	}

	localPath := filepath.Join(os.TempDir(), basename+".sql.gz")
	defer os.Remove(localPath)
	if err := sshCli.DownloadFile(waitCtx, remotePath, localPath); err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "local_dump_ready", "local_path", localPath)

	objectKey := fmt.Sprintf("project_%d/instance_%d/%s.sql.gz", inst.ProjectID, inst.ID, basename)
	s.logBackupPhase(job.ID, "minio_upload_start", "object_key", objectKey)
	size, err := minioCli.UploadFile(ctx, objectKey, localPath, "application/gzip")
	if err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "minio_upload_done", "bytes", size, "object_key", objectKey)
	job.MinioBucket = minioCli.Bucket()
	job.MinioObject = objectKey
	job.FileSize = size
	job.CheckOK = true
	_ = sv
	return nil
}

func (s *MysqlBackupService) runXtrabackupUpload(ctx context.Context, inst *model.MysqlBackupInstance, pw string, job *model.MysqlBackupJob) error {
	sshCli, sv, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()
	s.logBackupPhase(job.ID, "ssh_connected", "server_id", inst.ServerID)

	dataDir := strings.TrimSuffix(strings.TrimSpace(inst.RemoteDataDir), "/")
	logDir := strings.TrimSuffix(strings.TrimSpace(inst.RemoteLogDir), "/")
	if dataDir == "" || logDir == "" {
		return constants.ErrBadRequestWithMsg("xtrabackup 须配置 remote_data_dir 与 remote_log_dir")
	}

	startedAt := time.Now().UTC()
	basename, err := s.backupArtifactBasename(ctx, inst, startedAt)
	if err != nil {
		return err
	}
	remoteArchive := filepath.ToSlash(filepath.Join(dataDir, basename+".tar.gz"))
	logPath := filepath.ToSlash(filepath.Join(logDir, basename+".log"))
	job.RemotePath = remoteArchive
	job.BackupMode = model.MysqlBackupExecXtrabackup

	script := mysqlbackup.BuildXtrabackupRemoteScript(mysqlbackup.XtrabackupRemoteScriptParams{
		DataDir: dataDir, LogDir: logDir, Basename: basename,
		MySQLHost: inst.MysqlHost, MySQLPort: inst.MysqlPort, MySQLUser: inst.MysqlUser,
		MySQLPass: shellQuote(pw), MySQLDir: inst.MysqlDataDir, Parallel: 4, ShellQuote: shellQuote,
	})
	stopPoll := s.startPollBackupJobLog(ctx, job.ID, sshCli, logPath, "")
	defer stopPoll()

	s.logBackupPhase(job.ID, "xtrabackup_start",
		"data_dir", dataDir,
		"basename", basename,
		"archive", remoteArchive,
	)
	res, err := sshCli.Exec(ctx, script, 131072)
	job.LogExcerpt = mysqlbackup.TruncateLog(strings.TrimSpace(res.Stdout + res.Stderr))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("xtrabackup failed: %s", strings.TrimSpace(res.Stderr+res.Stdout))
	}
	archSize, sizeErr := sshCli.RemoteFileSize(remoteArchive)
	if sizeErr != nil {
		return fmt.Errorf("backup archive not found after xtrabackup: %w", sizeErr)
	}
	if err := mysqlbackup.ValidateArchiveSize(archSize, remoteArchive); err != nil {
		return err
	}
	s.logBackupPhase(job.ID, "xtrabackup_done", "exit_code", res.ExitCode, "archive_bytes", archSize, "ssh_duration_ms", res.Duration.Milliseconds())

	if !inst.UploadToMinio {
		job.CheckOK = true
		s.logBackupPhase(job.ID, "skip_minio_upload", "remote_archive", remoteArchive, "remote_log", logPath)
		_ = sv
		return nil
	}

	minioCli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return err
	}
	dlCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	localPath := filepath.Join(os.TempDir(), basename+".tar.gz")
	defer os.Remove(localPath)
	if err := sshCli.DownloadFile(dlCtx, remoteArchive, localPath); err != nil {
		return err
	}
	objectKey := fmt.Sprintf("project_%d/instance_%d/%s.tar.gz", inst.ProjectID, inst.ID, basename)
	s.logBackupPhase(job.ID, "minio_upload_start", "object_key", objectKey)
	size, err := minioCli.UploadFile(ctx, objectKey, localPath, "application/gzip")
	if err != nil {
		return err
	}
	job.MinioBucket = minioCli.Bucket()
	job.MinioObject = objectKey
	job.FileSize = size
	job.CheckOK = true
	_ = sv
	return nil
}

func (s *MysqlBackupService) backupArtifactNamePrefix(ctx context.Context, inst *model.MysqlBackupInstance) (string, error) {
	projectName := fmt.Sprintf("project_%d", inst.ProjectID)
	if proj, err := s.projectRepo.GetByID(ctx, inst.ProjectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}
	return mysqlbackup.BuildArtifactNamePrefix(projectName, inst.MysqlHost, inst.MysqlPort), nil
}

func (s *MysqlBackupService) ListJobs(ctx context.Context, q MysqlBackupJobListQuery) (*pagination.Result[model.MysqlBackupJob], error) {
	list, total, err := s.backupRepo.ListJobs(ctx, repository.MysqlBackupJobListParams{
		ProjectID: q.ProjectID, InstanceID: q.InstanceID, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, svcerr.Pass(ctx, "mysql.backup", "ListJobs", err)
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	return &pagination.Result[model.MysqlBackupJob]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *MysqlBackupService) PresignDownload(ctx context.Context, projectID, jobID uint) (string, error) {
	job, err := s.backupRepo.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", constants.ErrNotFound
		}
		return "", err
	}
	if job.ProjectID != projectID || job.Status != "success" {
		return "", constants.ErrBadRequestWithMsg("任务未完成")
	}
	if strings.TrimSpace(job.MinioObject) == "" {
		return "", constants.ErrBadRequestWithMsg("该任务未上传 MinIO，请查看日志中的远端路径")
	}
	cli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return "", err
	}
	return cli.PresignedGetURL(ctx, job.MinioObject, 15*time.Minute)
}

func (s *MysqlBackupService) backupArtifactBasename(ctx context.Context, inst *model.MysqlBackupInstance, at time.Time) (string, error) {
	projectName := fmt.Sprintf("project_%d", inst.ProjectID)
	if proj, err := s.projectRepo.GetByID(ctx, inst.ProjectID); err == nil && proj != nil {
		if n := strings.TrimSpace(proj.Name); n != "" {
			projectName = n
		}
	}
	return mysqlbackup.BuildArtifactBasename(projectName, inst.MysqlHost, inst.MysqlPort, at), nil
}

func (s *MysqlBackupService) startPollBackupJobLog(ctx context.Context, jobID uint, sshCli *sshclient.Client, logPath, prefix string) context.CancelFunc {
	pollCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				tail, err := s.tailRemoteFile(pollCtx, sshCli, logPath, 100)
				if err != nil {
					continue
				}
				excerpt := mysqlbackup.TruncateLog(prefix + strings.TrimSpace(tail))
				_ = s.backupRepo.PatchJob(pollCtx, jobID, map[string]any{"log_excerpt": excerpt})
			}
		}
	}()
	return cancel
}

func (s *MysqlBackupService) decryptInstancePassword(inst *model.MysqlBackupInstance) (string, error) {
	if inst == nil || inst.EncPassword == "" {
		return "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码，无法执行 mysqldump 回退")
	}
	return cryptox.DecryptString(s.aead, inst.EncPassword)
}

func (s *MysqlBackupService) loadInstanceSecrets(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupInstance, string, error) {
	inst, err := s.backupRepo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", constants.ErrNotFound
		}
		return nil, "", svcerr.Pass(ctx, "mysql.backup", "loadInstanceSecrets", err)
	}
	if inst.EncPassword == "" {
		return nil, "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码")
	}
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, "", svcerr.Pass(ctx, "mysql.backup", "loadInstanceSecrets", err)
	}
	return inst, pw, nil
}

func (s *MysqlBackupService) dialServer(ctx context.Context, serverID uint) (*sshclient.Client, *model.Server, error) {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, constants.ErrLogSourceServerNotFound
		}
		return nil, nil, svcerr.Pass(ctx, "mysql.backup", "dialServer", err)
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, serverID)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
	}
	cfg, err := s.decryptCredentialToSSHConfig(ctx, *sv, *cred)
	if err != nil {
		return nil, nil, err
	}
	cli, err := sshclient.Dial(ctx, cfg)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	return cli, sv, nil
}

func (s *MysqlBackupService) decryptCredentialToSSHConfig(ctx context.Context, sv model.Server, cred model.ServerCredential) (sshclient.Config, error) {
	cfg := sshclient.Config{Host: sv.Host, Port: sv.Port, Username: cred.Username}
	switch strings.ToLower(strings.TrimSpace(cred.AuthType)) {
	case "password":
		if cred.EncPassword == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg666b6d7186e5)
		}
		pw, err := cryptox.DecryptString(s.aead, *cred.EncPassword)
		if err != nil {
			return sshclient.Config{}, svcerr.Pass(ctx, "mysql.backup", "decryptCredentialToSSHConfig", err)
		}
		cfg.AuthType = sshclient.AuthPassword
		cfg.Password = pw
	case "key":
		if cred.EncPrivateKey == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg298c7d5f0d54)
		}
		pk, err := cryptox.DecryptString(s.aead, *cred.EncPrivateKey)
		if err != nil {
			return sshclient.Config{}, svcerr.Pass(ctx, "mysql.backup", "decryptCredentialToSSHConfig", err)
		}
		cfg.AuthType = sshclient.AuthKey
		cfg.PrivateKey = pk
		if cred.EncPassphrase != nil {
			pp, err := cryptox.DecryptString(s.aead, *cred.EncPassphrase)
			if err != nil {
				return sshclient.Config{}, svcerr.Pass(ctx, "mysql.backup", "decryptCredentialToSSHConfig", err)
			}
			cfg.Passphrase = pp
		}
	default:
		return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsge9e731f82ff9)
	}
	return cfg, nil
}

func (s *MysqlBackupService) ensureServerInProject(ctx context.Context, projectID, serverID uint) error {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrLogSourceServerNotFound
		}
		return svcerr.Pass(ctx, "mysql.backup", "ensureServerInProject", err)
	}
	if sv.ProjectID != projectID {
		return constants.ErrServerNotInCurrentProject
	}
	return nil
}

func (s *MysqlBackupService) toInstanceItem(ctx context.Context, inst model.MysqlBackupInstance) MysqlBackupInstanceItem {
	item := MysqlBackupInstanceItem{
		ID: inst.ID, ProjectID: inst.ProjectID, ServerID: inst.ServerID,
		Name: inst.Name, Enabled: inst.Enabled, MysqlHost: inst.MysqlHost, MysqlPort: inst.MysqlPort,
		MysqlUser: inst.MysqlUser, BackupMode: inst.BackupMode,
		BackupScope: inst.BackupScope, DatabaseName: inst.DatabaseName, TableName: inst.BackupTable,
		DatabaseNames: inst.DatabaseNames, RemoteDataDir: inst.RemoteDataDir, RemoteLogDir: inst.RemoteLogDir,
		MysqlDataDir: inst.MysqlDataDir,
		UploadToMinio: inst.UploadToMinio, MysqldumpWorkDir: inst.MysqldumpWorkDir, MysqldumpExtraArgs: inst.MysqldumpExtraArgs,
		ScheduleEnabled: inst.ScheduleEnabled, CronSpec: inst.CronSpec,
	}
	item.MysqldumpOptions = parseMysqldumpOptionsForAPI(inst.MysqldumpOptions)
	if inst.LastScheduledAt != nil && !inst.LastScheduledAt.IsZero() {
		item.LastScheduledAt = inst.LastScheduledAt.Format(time.RFC3339)
	}
	if sv, err := s.serverRepo.GetByID(ctx, inst.ServerID); err == nil {
		item.ServerName = sv.Name
	}
	if !inst.CreatedAt.IsZero() {
		item.CreatedAt = inst.CreatedAt.Format(time.RFC3339)
	}
	if !inst.UpdatedAt.IsZero() {
		item.UpdatedAt = inst.UpdatedAt.Format(time.RFC3339)
	}
	return item
}

func (s *MysqlBackupService) ListMysqldumpOptions() []mysqlbackup.MysqldumpOption {
	return mysqlbackup.MysqldumpOptionCatalog
}

func marshalMysqldumpOptionIDs(ids []string) string {
	if len(ids) == 0 {
		bs, _ := json.Marshal(mysqlbackup.DefaultMysqldumpOptionIDs())
		return string(bs)
	}
	bs, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(bs)
}

func parseMysqldumpOptionsForAPI(raw string) []string {
	ids, err := mysqlbackup.ParseMysqldumpOptionIDs(raw)
	if err != nil {
		return mysqlbackup.DefaultMysqldumpOptionIDs()
	}
	return ids
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (s *MysqlBackupService) tailRemoteFile(ctx context.Context, sshCli *sshclient.Client, path string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	script := fmt.Sprintf(`tail -n %d %q 2>/dev/null || true`, lines, path)
	res, err := sshCli.Exec(ctx, script, 65536)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout + res.Stderr), nil
}
