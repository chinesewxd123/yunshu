package mysqlbackup

import "fmt"

// XtrabackupRemoteScriptParams 远端 xtrabackup 热备脚本（须已安装 xtrabackup 2.4+ 或 innobackupex）。
type XtrabackupRemoteScriptParams struct {
	DataDir   string
	LogDir    string
	Basename  string
	MySQLHost string
	MySQLPort int
	MySQLUser string
	MySQLPass string // 已 shell 转义
	Parallel  int
	ShellQuote func(string) string
}

// BuildXtrabackupRemoteScript 执行备份 → prepare → 打包为 ${basename}.tar.gz，日志 ${basename}.log。
func BuildXtrabackupRemoteScript(p XtrabackupRemoteScriptParams) string {
	q := p.ShellQuote
	if p.Parallel <= 0 {
		p.Parallel = 4
	}
	dataDir := q(p.DataDir)
	logDir := q(p.LogDir)
	tmpDir := q(p.DataDir + "/." + p.Basename + ".tmp")
	archive := q(p.DataDir + "/" + p.Basename + ".tar.gz")
	logPath := q(p.LogDir + "/" + p.Basename + ".log")
	return fmt.Sprintf(`set -euo pipefail
if command -v xtrabackup >/dev/null 2>&1; then XB=xtrabackup
elif command -v innobackupex >/dev/null 2>&1; then XB=innobackupex
else echo "xtrabackup/innobackupex 未安装"; exit 127
fi
mkdir -p %s %s
export MYSQL_PWD=%s
LOG=%s
ARCHIVE=%s
TMP=%s
exec > >(tee -a "$LOG") 2>&1
echo "[$(date '+%%F %%T')] xtrabackup start host=%s port=%d user=%s basename=%s"
rm -rf "$TMP"
"$XB" --backup --host=%s --port=%d --user=%s --target-dir="$TMP" --parallel=%d
"$XB" --prepare --target-dir="$TMP"
if command -v pigz >/dev/null 2>&1; then
  tar -I "pigz -1" -cf "$ARCHIVE" -C "$TMP" .
else
  tar -I "gzip -1" -cf "$ARCHIVE" -C "$TMP" .
fi
rm -rf "$TMP"
SZ=$(stat -c%%s "$ARCHIVE" 2>/dev/null || echo 0)
echo "[$(date '+%%F %%T')] archive $ARCHIVE size=$SZ bytes"
echo "completed OK!"
tail -n 80 "$LOG" 2>/dev/null || true
`,
		dataDir, logDir, p.MySQLPass, logPath, archive, tmpDir,
		p.MySQLHost, p.MySQLPort, p.MySQLUser, p.Basename,
		q(p.MySQLHost), p.MySQLPort, q(p.MySQLUser), p.Parallel,
	)
}
