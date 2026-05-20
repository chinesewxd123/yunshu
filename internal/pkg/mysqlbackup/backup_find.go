package mysqlbackup

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BackupArtifact 按统一命名规则定位的备份产物。
type BackupArtifact struct {
	BackupFile string
	LogFile    string
	Basename   string
	OK         bool
	Message    string
	Stdout     string
}

const defaultBackupLookbackCount = 30

// BuildFindLatestBackupScript 在 data 目录按文件名前缀查找最新有效备份（*.tar.gz + 同名 .log 末行 completed OK!）。
func BuildFindLatestBackupScript(dataDir, logDir, namePrefix string, maxCandidates int) string {
	if maxCandidates <= 0 {
		maxCandidates = defaultBackupLookbackCount
	}
	dataDir = strings.TrimSpace(dataDir)
	logDir = strings.TrimSpace(logDir)
	namePrefix = strings.TrimSpace(namePrefix)
	return fmt.Sprintf(`set -e
DATA_DIR=%q
LOG_DIR=%q
PREFIX=%q
found=0
for f in $(ls -1t "$DATA_DIR"/${PREFIX}*.tar.gz 2>/dev/null | head -n %d); do
  [ -f "$f" ] || continue
  bn=$(basename "$f" .tar.gz)
  log="$LOG_DIR/${bn}.log"
  if [ ! -f "$log" ]; then
    echo "SKIP|${f}|missing_log|${log}"
    continue
  fi
  last=$(tail -n 1 "$log" 2>/dev/null || true)
  if echo "$last" | grep -Fq '%s'; then
    echo "OK|${f}|${log}|${bn}"
    found=1
    break
  fi
  echo "SKIP|${f}|log_incomplete|${last}"
done
if [ "$found" -eq 0 ]; then
  echo NOT_FOUND
  exit 2
fi
`, dataDir, logDir, namePrefix, maxCandidates, BackupCompletedMarker)
}

// BackupPathsForBasename 由基名生成数据包与日志路径。
func BackupPathsForBasename(dataDir, logDir, basename string) (backupFile, logFile string) {
	basename = strings.TrimSpace(basename)
	backupFile = filepath.ToSlash(filepath.Join(dataDir, basename+".tar.gz"))
	logFile = filepath.ToSlash(filepath.Join(logDir, basename+".log"))
	return backupFile, logFile
}

// ParseFindLatestBackupOutput 解析 BuildFindLatestBackupScript 输出。
func ParseFindLatestBackupOutput(stdout string, port int) BackupArtifact {
	stdout = strings.TrimSpace(stdout)
	out := BackupArtifact{Stdout: stdout}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OK|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				out.BackupFile = parts[1]
				out.LogFile = parts[2]
				out.Basename = parts[3]
				out.OK = true
				out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=1i", port)
				return out
			}
		}
	}
	if strings.Contains(stdout, "NOT_FOUND") {
		out.Message = fmt.Sprintf("未找到有效备份（命名前缀匹配且日志末行含 %s）", BackupCompletedMarker)
		return out
	}
	out.Message = fmt.Sprintf("mysqlbackupcheck,port=%d status=0i", port)
	return out
}
