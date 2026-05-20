package mysqlbackup

// MaxBackupLogBytes 写入任务记录的日志上限。
const MaxBackupLogBytes = 32 * 1024

// TruncateLog 截断过长日志，避免 DB 字段膨胀。
func TruncateLog(s string) string {
	if len(s) <= MaxBackupLogBytes {
		return s
	}
	return s[len(s)-MaxBackupLogBytes:]
}
