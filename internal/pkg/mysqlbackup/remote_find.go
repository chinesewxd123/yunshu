package mysqlbackup

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// RemoteBackupArtifact 远端 xtrabackup 产物定位结果。
type RemoteBackupArtifact struct {
	BackupFile string
	LogFile    string
	BackupDay  string // YYYY-MM-DD
	OK         bool
	Message    string
	Stdout     string
}

const defaultRemoteBackupLookbackDays = 30

// BackupPathsForDay 按日期生成 full_YYYYMMDD.tar.gz 与日志路径（本地/远端规则一致）。
func BackupPathsForDay(dataDir, logDir string, day time.Time) (backupFile, logFile string) {
	ymd := day.Format("20060102")
	backupFile = filepath.ToSlash(filepath.Join(dataDir, fmt.Sprintf("full_%s.tar.gz", ymd)))
	logFile = filepath.ToSlash(filepath.Join(logDir, fmt.Sprintf("full_backup_data_%s.log", day.Format("2006-01-02"))))
	return backupFile, logFile
}

// BuildFindLatestRemoteBackupScript 在远端 data 目录按修改时间查找最近的有效 full_*.tar.gz（日志末行含 completed OK!）。
func BuildFindLatestRemoteBackupScript(dataDir, logDir string, maxCandidates int) string {
	if maxCandidates <= 0 {
		maxCandidates = 30
	}
	dataDir = strings.TrimSpace(dataDir)
	logDir = strings.TrimSpace(logDir)
	return fmt.Sprintf(`set -e
DATA_DIR=%q
LOG_DIR=%q
found=0
for f in $(ls -1t "$DATA_DIR"/full_*.tar.gz 2>/dev/null | head -n %d); do
  [ -f "$f" ] || continue
  bn=$(basename "$f" .tar.gz)
  day=${bn#full_}
  if [ ${#day} -ne 8 ]; then continue; fi
  y=${day:0:4}; m=${day:4:2}; d=${day:6:2}
  log="$LOG_DIR/full_backup_data_${y}-${m}-${d}.log"
  if [ ! -f "$log" ]; then
    echo "SKIP|${f}|missing_log|${log}"
    continue
  fi
  last=$(tail -n 1 "$log" 2>/dev/null || true)
  if echo "$last" | grep -Fq 'completed OK!'; then
    echo "OK|${f}|${log}|${y}-${m}-${d}"
    found=1
    break
  fi
  echo "SKIP|${f}|log_incomplete|${last}"
done
if [ "$found" -eq 0 ]; then
  echo NOT_FOUND
  exit 2
fi
`, dataDir, logDir, maxCandidates)
}

// ParseFindLatestRemoteBackupOutput 解析 BuildFindLatestRemoteBackupScript 输出。
func ParseFindLatestRemoteBackupOutput(stdout string, port int) RemoteBackupArtifact {
	stdout = strings.TrimSpace(stdout)
	out := RemoteBackupArtifact{Stdout: stdout}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OK|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				out.BackupFile = parts[1]
				out.LogFile = parts[2]
				out.BackupDay = parts[3]
				out.OK = true
				out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=1i", port)
				return out
			}
		}
	}
	if strings.Contains(stdout, "NOT_FOUND") {
		out.Message = fmt.Sprintf("近 %d 天内未找到有效的 xtrabackup 备份包（full_*.tar.gz 且日志 completed OK!）", defaultRemoteBackupLookbackDays)
		return out
	}
	out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=0i", port)
	return out
}
