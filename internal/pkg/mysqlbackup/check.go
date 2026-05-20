package mysqlbackup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RemoteCheckResult 远端 xtrabackup 备份检查结果（对齐 mysql_golang_tools mysqlbackupcheck）。
type RemoteCheckResult struct {
	OK           bool   `json:"ok"`
	BackupFile   string `json:"backup_file,omitempty"`
	LogFile      string `json:"log_file,omitempty"`
	LogCompleted bool   `json:"log_completed"`
	Message      string `json:"message,omitempty"`
}

// CheckRemoteBackupFile 检查昨日 full_YYYYMMDD.tar.gz 及日志 completed OK!（本地路径，供单测）。
func CheckRemoteBackupFile(dataDir, logDir string, port int, day time.Time) RemoteCheckResult {
	filename := fmt.Sprintf("full_%s.tar.gz", day.Format("20060102"))
	filePath := filepath.Join(dataDir, filename)
	logPath := filepath.Join(logDir, fmt.Sprintf("full_backup_data_%s.log", day.Format("2006-01-02")))
	return checkPaths(filePath, logPath, port)
}

func checkPaths(filePath, logPath string, port int) RemoteCheckResult {
	res := RemoteCheckResult{BackupFile: filePath, LogFile: logPath}
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			res.Message = fmt.Sprintf("未找到备份文件: %s", filePath)
		} else {
			res.Message = err.Error()
		}
		return res
	}
	res.LogCompleted = logFileCompletedOK(logPath)
	res.OK = res.LogCompleted
	if !res.OK {
		res.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=0i", port)
	} else {
		res.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=1i", port)
	}
	return res
}

func logFileCompletedOK(logFilePath string) bool {
	file, err := os.Open(logFilePath)
	if err != nil {
		return false
	}
	defer file.Close()
	var lastLine string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return false
	}
	if strings.Contains(lastLine, BackupCompletedMarker) {
		return true
	}
	// 兼容旧脚本末行（无 xtrabackup 误匹配时 archive 行在 completed 之前）
	return strings.Contains(lastLine, "completed OK!") && strings.Contains(lastLine, "archive=")
}

// ParseRemoteCheckOutput 解析 SSH 上 check 脚本输出。
func ParseRemoteCheckOutput(stdout string) bool {
	return strings.Contains(stdout, "status=1i")
}
