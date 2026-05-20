package model

import "strings"

// IsXtrabackupBackupMode 物理备份（经 SSH 执行 xtrabackup）；兼容历史值 remote_check。
func IsXtrabackupBackupMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case MysqlBackupModeXtrabackup, MysqlBackupModeRemoteCheck:
		return true
	default:
		return false
	}
}
