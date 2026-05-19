package mysqlbackup

import "fmt"

// MysqldumpRemoteScriptParams 远端 mysqldump 脚本参数（Linux bash）。
type MysqldumpRemoteScriptParams struct {
	WorkDir     string
	Basename    string
	MySQLHost   string
	MySQLPort   int
	MySQLUser   string
	MySQLPass   string // 已 shell 转义
	DumpFlags   string
	DumpTarget  string // 已格式化目标库表参数
	ShellQuote  func(string) string
}

// BuildMysqldumpRemoteScript 执行 mysqldump，stderr/进度写入 .log，数据写入 .sql.gz。
func BuildMysqldumpRemoteScript(p MysqldumpRemoteScriptParams) string {
	q := p.ShellQuote
	workDir := q(p.WorkDir)
	logPath := q(p.WorkDir + "/" + p.Basename + ".log")
	sqlPath := q(p.WorkDir + "/" + p.Basename + ".sql.gz")
	return fmt.Sprintf(`set -euo pipefail
mkdir -p %s
LOG=%s
SQL=%s
export MYSQL_PWD=%s
echo "[$(date '+%%F %%T')] mysqldump start host=%s port=%d user=%s -> $SQL" > "$LOG"
( while sleep 15; do echo "[$(date '+%%F %%T')] progress sql.gz $(stat -c%%s "$SQL" 2>/dev/null || echo 0) bytes"; done >>"$LOG" ) &
MON=$!
mysqldump -h%s -P%d -u%s %s %s --verbose 2>>"$LOG" | gzip -c > "$SQL"
EC=$?
kill "$MON" 2>/dev/null || true
wait "$MON" 2>/dev/null || true
echo "[$(date '+%%F %%T')] mysqldump exit=$EC sql.gz $(stat -c%%s "$SQL" 2>/dev/null || echo 0) bytes" >>"$LOG"
tail -n 120 "$LOG" 2>/dev/null || true
exit $EC
`,
		workDir, logPath, sqlPath, p.MySQLPass,
		p.MySQLHost, p.MySQLPort, p.MySQLUser,
		q(p.MySQLHost), p.MySQLPort, q(p.MySQLUser), p.DumpFlags, p.DumpTarget,
	)
}
