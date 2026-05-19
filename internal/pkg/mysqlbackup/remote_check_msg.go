package mysqlbackup

import (
	"fmt"
	"strings"
)

// RemoteCheckFallbackLogPrefix 远端 xtrabackup 未命中时写入任务日志的说明（平台不执行 xtrabackup，仅扫描已有产物）。
func RemoteCheckFallbackLogPrefix(dataDir, logDir, mysqldumpWorkDir, findDetail string) string {
	var b strings.Builder
	b.WriteString("[未找到远端 xtrabackup 产物]\n")
	b.WriteString("说明：本平台不会执行 xtrabackup，仅扫描远端目录中 cron/脚本已生成的 full_*.tar.gz（对应日志末行须含 completed OK!）。\n")
	b.WriteString(fmt.Sprintf("数据目录: %s\n", strings.TrimSpace(dataDir)))
	b.WriteString(fmt.Sprintf("日志目录: %s\n", strings.TrimSpace(logDir)))
	b.WriteString(fmt.Sprintf("已自动改用 mysqldump，落盘目录: %s\n\n", strings.TrimSpace(mysqldumpWorkDir)))
	if strings.TrimSpace(findDetail) != "" {
		b.WriteString(strings.TrimSpace(findDetail))
		b.WriteString("\n\n")
	}
	return b.String()
}
