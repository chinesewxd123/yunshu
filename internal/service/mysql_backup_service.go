package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/mysqlbackup"
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
	infoLog      *slog.Logger
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
	}, nil
}

func (s *MysqlBackupService) SetInfoLogger(log *slog.Logger) {
	s.infoLog = log
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
	RemoteDataDir string `json:"remote_data_dir"`
	RemoteLogDir  string `json:"remote_log_dir"`
	ScheduleEnabled bool   `json:"schedule_enabled"`
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
	RemoteDataDir string `json:"remote_data_dir"`
	RemoteLogDir  string `json:"remote_log_dir"`
	ScheduleEnabled *bool  `json:"schedule_enabled"`
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
		return nil, svcerr.Pass("mysql.backup", "ListInstances", err)
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
	if mode != model.MysqlBackupModeMysqldump && mode != model.MysqlBackupModeRemoteCheck {
		return nil, constants.ErrBadRequestWithMsg("backup_mode 须为 mysqldump 或 remote_check")
	}

	var inst *model.MysqlBackupInstance
	if id > 0 {
		existing, err := s.backupRepo.GetInstanceInProject(ctx, req.ProjectID, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFound
			}
			return nil, svcerr.Pass("mysql.backup", "UpsertInstance", err)
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
			return nil, constants.ErrBadRequestWithMsg("remote_check 模式须填写 remote_data_dir 与 remote_log_dir")
		}
	}
	inst.DatabaseNames = strings.TrimSpace(req.DatabaseNames)
	inst.RemoteDataDir = strings.TrimSpace(req.RemoteDataDir)
	inst.RemoteLogDir = strings.TrimSpace(req.RemoteLogDir)

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
			return nil, svcerr.Pass("mysql.backup", "UpsertInstance", err)
		}
		inst.EncPassword = enc
	} else if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("新建实例须填写 mysql_password")
	}

	if id > 0 {
		if err := s.backupRepo.UpdateInstance(ctx, inst); err != nil {
			return nil, svcerr.Pass("mysql.backup", "UpsertInstance", err)
		}
	} else {
		if err := s.backupRepo.CreateInstance(ctx, inst); err != nil {
			return nil, svcerr.Pass("mysql.backup", "UpsertInstance", err)
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
		return svcerr.Pass("mysql.backup", "DeleteInstance", err)
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

func (s *MysqlBackupService) CheckRemoteBackup(ctx context.Context, projectID, instanceID uint, dayOffset int) (*mysqlbackup.RemoteCheckResult, error) {
	inst, _, err := s.loadInstanceSecrets(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.BackupMode != model.MysqlBackupModeRemoteCheck {
		return nil, constants.ErrBadRequestWithMsg("该实例不是 remote_check 模式")
	}
	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return nil, err
	}
	defer sshCli.Close()

	day := time.Now().AddDate(0, 0, dayOffset)
	filename := fmt.Sprintf("full_%s.tar.gz", day.Format("20060102"))
	remoteFile := filepath.ToSlash(filepath.Join(inst.RemoteDataDir, filename))
	logFile := filepath.ToSlash(filepath.Join(inst.RemoteLogDir, fmt.Sprintf("full_backup_data_%s.log", day.Format("2006-01-02"))))

	checkScript := fmt.Sprintf(
		`if [ ! -f %q ]; then echo MISSING_BACKUP_FILE; exit 2; fi; if [ ! -f %q ]; then echo MISSING_LOG_FILE; exit 3; fi; tail -n 1 %q`,
		remoteFile, logFile, logFile,
	)
	res, err := sshCli.Exec(ctx, checkScript, 8192)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHExecFailedPrefix + err.Error())
	}
	stdout := strings.TrimSpace(res.Stdout)
	logOK := strings.Contains(stdout, "completed OK!")
	out := &mysqlbackup.RemoteCheckResult{
		BackupFile:   remoteFile,
		LogFile:      logFile,
		LogCompleted: logOK,
		OK:           res.ExitCode == 0 && logOK,
	}
	dayLabel := day.Format("2006-01-02")
	if out.OK {
		out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=1i", inst.MysqlPort)
	} else {
		switch {
		case strings.Contains(stdout, "MISSING_BACKUP_FILE"):
			out.Message = fmt.Sprintf("未找到昨日(%s)备份包 %s（请确认远端 xtrabackup 已落盘且目录正确）", dayLabel, remoteFile)
		case strings.Contains(stdout, "MISSING_LOG_FILE"):
			out.Message = fmt.Sprintf("未找到日志 %s（期望 full_backup_data_%s.log）", logFile, day.Format("2006-01-02"))
		case stdout != "" && !logOK:
			out.Message = fmt.Sprintf("日志末行未含 completed OK!（%s 最后一行: %s）", logFile, stdout)
		default:
			out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=0i（检查 %s）", inst.MysqlPort, remoteFile)
		}
	}
	return out, nil
}

func (s *MysqlBackupService) RunBackup(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupJob, error) {
	return s.enqueueBackup(ctx, projectID, instanceID, model.MysqlBackupTriggerManual)
}

func (s *MysqlBackupService) enqueueBackup(ctx context.Context, projectID, instanceID uint, trigger string) (*model.MysqlBackupJob, error) {
	n, _ := s.backupRepo.FailStaleRunningJobs(ctx, 2*time.Hour)
	if n > 0 && s.infoLog != nil {
		s.infoLog.Warn("mysql_backup_stale_jobs_marked_failed", slog.Int64("count", n))
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
		return nil, svcerr.Pass("mysql.backup", "enqueueBackup", err)
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
		return nil, svcerr.Pass("mysql.backup", "enqueueBackup", err)
	}

	go s.runBackupJobAsync(job.ID, projectID, instanceID, trigger)
	return job, nil
}

const mysqlBackupJobTimeout = 35 * time.Minute

func (s *MysqlBackupService) runBackupJobAsync(jobID, projectID, instanceID uint, trigger string) {
	ctx, cancel := context.WithTimeout(context.Background(), mysqlBackupJobTimeout)
	defer cancel()
	s.finishBackupJob(ctx, jobID, projectID, instanceID, trigger)
}

func (s *MysqlBackupService) finishBackupJob(ctx context.Context, jobID, projectID, instanceID uint, trigger string) {
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
		return
	}
	target := mysqlbackup.BuildDumpTarget(inst)

	var runErr error
	switch inst.BackupMode {
	case model.MysqlBackupModeRemoteCheck:
		runErr = s.runRemoteCheckUpload(ctx, inst, job)
	default:
		runErr = s.runMysqldumpUpload(ctx, inst, pw, job, target)
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
	if trigger == model.MysqlBackupTriggerScheduled && s.infoLog != nil {
		if runErr != nil {
			s.infoLog.Warn("mysql_backup_scheduled_failed",
				slog.Uint64("instance_id", uint64(inst.ID)),
				slog.String("name", inst.Name),
				slog.String("error", runErr.Error()),
			)
		} else {
			s.infoLog.Info("mysql_backup_scheduled_ok",
				slog.Uint64("instance_id", uint64(inst.ID)),
				slog.String("name", inst.Name),
				slog.String("object", job.MinioObject),
			)
		}
	}
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

func (s *MysqlBackupService) runMysqldumpUpload(ctx context.Context, inst *model.MysqlBackupInstance, pw string, job *model.MysqlBackupJob, target mysqlbackup.DumpTarget) error {
	minioCli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return err
	}
	sshCli, sv, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()

	ts := time.Now().UTC().Format("20060102_150405")
	remotePath := fmt.Sprintf("/tmp/yunshu_mysql_%d_%s.sql.gz", inst.ID, ts)
	job.RemotePath = remotePath

	dumpTarget := mysqlbackup.FormatDumpArgsShell(target, shellQuote)
	label := target.ObjectLabel
	if label == "" {
		label = "dump"
	}

	logPath := fmt.Sprintf("/tmp/yunshu_mysql_%d_%s.log", inst.ID, ts)
	escapedPW := shellQuote(pw)
	dumpCmd := fmt.Sprintf(
		`set -euo pipefail; LOG=%s; export MYSQL_PWD=%s; mysqldump -h%s -P%d -u%s --single-transaction --quick --routines --triggers %s 2>"$LOG" | gzip -c > %s; EC=$?; echo "=== mysqldump exit=$EC ==="; tail -n 120 "$LOG" 2>/dev/null || true; exit $EC`,
		shellQuote(logPath), escapedPW, shellQuote(inst.MysqlHost), inst.MysqlPort, shellQuote(inst.MysqlUser), dumpTarget, shellQuote(remotePath),
	)
	res, err := sshCli.Exec(ctx, dumpCmd, 65536)
	job.LogExcerpt = mysqlbackup.TruncateLog(strings.TrimSpace(res.Stdout + res.Stderr))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(res.Stderr+res.Stdout))
	}
	_ = sshCli.RemoveRemoteFile(logPath)

	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err := sshCli.WaitRemoteFile(waitCtx, remotePath, 1024, 30*time.Minute); err != nil {
		return err
	}

	localPath := filepath.Join(os.TempDir(), fmt.Sprintf("mysql_bak_%d_%s.gz", inst.ID, ts))
	defer os.Remove(localPath)
	if err := sshCli.DownloadFile(waitCtx, remotePath, localPath); err != nil {
		return err
	}
	_ = sshCli.RemoveRemoteFile(remotePath)

	objectKey := fmt.Sprintf("project_%d/instance_%d/%s_%s.sql.gz", inst.ProjectID, inst.ID, label, ts)
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

func (s *MysqlBackupService) runRemoteCheckUpload(ctx context.Context, inst *model.MysqlBackupInstance, job *model.MysqlBackupJob) error {
	check, err := s.CheckRemoteBackup(ctx, inst.ProjectID, inst.ID, -1)
	if err != nil {
		return err
	}
	job.CheckOK = check.OK
	job.RemotePath = check.BackupFile

	sshCli, _, err := s.dialServer(ctx, inst.ServerID)
	if err != nil {
		return err
	}
	defer sshCli.Close()
	if logTail, err := s.tailRemoteFile(ctx, sshCli, check.LogFile, 80); err == nil {
		job.LogExcerpt = mysqlbackup.TruncateLog(logTail)
	} else {
		job.LogExcerpt = check.Message
	}
	if !check.OK {
		return fmt.Errorf("远端备份检查未通过: %s", check.Message)
	}

	minioCli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102_150405")
	localPath := filepath.Join(os.TempDir(), fmt.Sprintf("mysql_remote_%d_%s.tar.gz", inst.ID, ts))
	defer os.Remove(localPath)
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	if err := sshCli.DownloadFile(waitCtx, check.BackupFile, localPath); err != nil {
		return err
	}
	objectKey := fmt.Sprintf("project_%d/instance_%d/remote_%s.tar.gz", inst.ProjectID, inst.ID, ts)
	size, err := minioCli.UploadFile(ctx, objectKey, localPath, "application/gzip")
	if err != nil {
		return err
	}
	job.MinioBucket = minioCli.Bucket()
	job.MinioObject = objectKey
	job.FileSize = size
	return nil
}

func (s *MysqlBackupService) ListJobs(ctx context.Context, q MysqlBackupJobListQuery) (*pagination.Result[model.MysqlBackupJob], error) {
	list, total, err := s.backupRepo.ListJobs(ctx, repository.MysqlBackupJobListParams{
		ProjectID: q.ProjectID, InstanceID: q.InstanceID, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, svcerr.Pass("mysql.backup", "ListJobs", err)
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
	if job.ProjectID != projectID || job.Status != "success" || strings.TrimSpace(job.MinioObject) == "" {
		return "", constants.ErrBadRequestWithMsg("任务未完成或无归档对象")
	}
	cli, err := objectstore.NewFromDB(ctx, s.db)
	if err != nil {
		return "", err
	}
	return cli.PresignedGetURL(ctx, job.MinioObject, 15*time.Minute)
}

func (s *MysqlBackupService) loadInstanceSecrets(ctx context.Context, projectID, instanceID uint) (*model.MysqlBackupInstance, string, error) {
	inst, err := s.backupRepo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", constants.ErrNotFound
		}
		return nil, "", svcerr.Pass("mysql.backup", "loadInstanceSecrets", err)
	}
	if inst.EncPassword == "" {
		return nil, "", constants.ErrBadRequestWithMsg("未配置 MySQL 密码")
	}
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, "", svcerr.Pass("mysql.backup", "loadInstanceSecrets", err)
	}
	return inst, pw, nil
}

func (s *MysqlBackupService) dialServer(ctx context.Context, serverID uint) (*sshclient.Client, *model.Server, error) {
	sv, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, constants.ErrLogSourceServerNotFound
		}
		return nil, nil, svcerr.Pass("mysql.backup", "dialServer", err)
	}
	cred, err := s.serverRepo.GetCredentialByServerID(ctx, serverID)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgfeb33ee7c48c)
	}
	cfg, err := s.decryptCredentialToSSHConfig(*sv, *cred)
	if err != nil {
		return nil, nil, err
	}
	cli, err := sshclient.Dial(ctx, cfg)
	if err != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(constants.ErrMsgSSHConnectFailedPrefix + err.Error())
	}
	return cli, sv, nil
}

func (s *MysqlBackupService) decryptCredentialToSSHConfig(sv model.Server, cred model.ServerCredential) (sshclient.Config, error) {
	cfg := sshclient.Config{Host: sv.Host, Port: sv.Port, Username: cred.Username}
	switch strings.ToLower(strings.TrimSpace(cred.AuthType)) {
	case "password":
		if cred.EncPassword == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg666b6d7186e5)
		}
		pw, err := cryptox.DecryptString(s.aead, *cred.EncPassword)
		if err != nil {
			return sshclient.Config{}, svcerr.Pass("mysql.backup", "decryptCredentialToSSHConfig", err)
		}
		cfg.AuthType = sshclient.AuthPassword
		cfg.Password = pw
	case "key":
		if cred.EncPrivateKey == nil {
			return sshclient.Config{}, constants.ErrBadRequestWithMsg(constants.ErrMsg298c7d5f0d54)
		}
		pk, err := cryptox.DecryptString(s.aead, *cred.EncPrivateKey)
		if err != nil {
			return sshclient.Config{}, svcerr.Pass("mysql.backup", "decryptCredentialToSSHConfig", err)
		}
		cfg.AuthType = sshclient.AuthKey
		cfg.PrivateKey = pk
		if cred.EncPassphrase != nil {
			pp, err := cryptox.DecryptString(s.aead, *cred.EncPassphrase)
			if err != nil {
				return sshclient.Config{}, svcerr.Pass("mysql.backup", "decryptCredentialToSSHConfig", err)
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
		return svcerr.Pass("mysql.backup", "ensureServerInProject", err)
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
		ScheduleEnabled: inst.ScheduleEnabled, CronSpec: inst.CronSpec,
	}
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
