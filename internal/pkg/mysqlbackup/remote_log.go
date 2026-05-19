package mysqlbackup

import (
	"fmt"
	"strings"
)

// BuildTailXtrabackupLogScript 按备份日尝试多种常见日志文件名并输出尾部（兼容不同 xtrabackup 脚本命名）。
func BuildTailXtrabackupLogScript(logDir, backupDay string, lines int) string {
	if lines <= 0 {
		lines = 120
	}
	logDir = stringsTrimSuffixSlash(logDir)
	backupDay = strings.TrimSpace(backupDay)
	return fmt.Sprintf(`set -e
LOG_DIR=%q
DAY=%q
LINES=%d
try_tail() {
  if [ -f "$1" ]; then
    echo "=== log file: $1 ==="
    tail -n "$LINES" "$1"
    return 0
  fi
  return 1
}
if [ -n "$DAY" ]; then
  y=${DAY%%-*}; m=${DAY#*-}; m=${m%%-*}; d=${DAY##*-}
  ymd="${y}${m}${d}"
  try_tail "$LOG_DIR/full_backup_data_${DAY}.log" && exit 0
  try_tail "$LOG_DIR/full_backup_${ymd}.log" && exit 0
  try_tail "$LOG_DIR/full_backup_data_${ymd}.log" && exit 0
fi
latest=$(ls -1t "$LOG_DIR"/*.log 2>/dev/null | head -n 1)
if [ -n "$latest" ]; then
  try_tail "$latest" && exit 0
fi
echo "=== no xtrabackup log found under $LOG_DIR (day=$DAY) ==="
exit 0
`, logDir, backupDay, lines)
}

func stringsTrimSuffixSlash(s string) string {
	return strings.TrimSuffix(strings.TrimSpace(s), "/")
}
