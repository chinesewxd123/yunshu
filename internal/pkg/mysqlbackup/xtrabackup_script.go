package mysqlbackup

import (
	"fmt"
	"strings"
)

// XtrabackupRemoteScriptParams 远端 xtrabackup 热备脚本（须在能访问 MySQL datadir 的机器上执行）。
type XtrabackupRemoteScriptParams struct {
	DataDir    string // 备份产物 .tar.gz 输出目录
	LogDir     string
	Basename   string
	MySQLHost  string
	MySQLPort  int
	MySQLUser  string
	MySQLPass  string // 已 shell 转义
	MySQLDir   string // 可选：MySQL datadir，留空则 SELECT @@datadir
	Parallel   int
	ShellQuote func(string) string
}

// BuildXtrabackupRemoteScript 执行备份 → prepare → 打包为 ${basename}.tar.gz，日志 ${basename}.log。
// 日志只写远端文件，不经 SSH stdout（避免 tee 塞满通道后 tar 卡死）。
func BuildXtrabackupRemoteScript(p XtrabackupRemoteScriptParams) string {
	q := p.ShellQuote
	if p.Parallel <= 0 {
		p.Parallel = 4
	}
	outDir := q(p.DataDir)
	logDir := q(p.LogDir)
	mysqlDirOverride := `""`
	if d := strings.TrimSpace(p.MySQLDir); d != "" {
		mysqlDirOverride = q(d)
	}
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
MYSQL_DIR_OVERRIDE=%s
: >"$LOG"
echo "[$(date '+%%F %%T')] xtrabackup start host=%s port=%d user=%s basename=%s" >>"$LOG"
if [ -n "$MYSQL_DIR_OVERRIDE" ]; then
  MYSQL_DATADIR="$MYSQL_DIR_OVERRIDE"
else
  MYSQL_DATADIR=$(mysql --no-defaults -h%s -P%d -u%s -Nse "SELECT @@datadir" 2>/dev/null | tr -d '\r\n')
fi
MYSQL_DATADIR="${MYSQL_DATADIR%%/}"
if [ -z "$MYSQL_DATADIR" ] || [ ! -d "$MYSQL_DATADIR" ]; then
  echo "ERROR: MySQL datadir 无效或不可访问: ${MYSQL_DATADIR:-<empty>}" >>"$LOG"
  echo "提示: xtrabackup 须在 datadir 所在主机执行；Docker 请在实例中填写宿主机 datadir" >>"$LOG"
  exit 1
fi
echo "[$(date '+%%F %%T')] using datadir=$MYSQL_DATADIR" >>"$LOG"
rm -rf "$TMP"
if [ "$XB" = "xtrabackup" ]; then
  "$XB" --no-defaults --backup \
    --datadir="$MYSQL_DATADIR" \
    --host=%s --port=%d --user=%s --password=%s \
    --target-dir="$TMP" --parallel=%d >>"$LOG" 2>&1
  "$XB" --prepare --target-dir="$TMP" >>"$LOG" 2>&1
else
  "$XB" --no-defaults --datadir="$MYSQL_DATADIR" \
    --host=%s --port=%d --user=%s --password=%s \
    --parallel=%d "$TMP" >>"$LOG" 2>&1
  "$XB" --prepare --target-dir="$TMP" >>"$LOG" 2>&1
fi
`+shellTarGzFromDir+`
rm -rf "$TMP"
SZ=$(stat -c%%s "$ARCHIVE" 2>/dev/null || echo 0)
echo "[$(date '+%%F %%T')] archive $ARCHIVE size=$SZ bytes" >>"$LOG"
echo "`+BackupCompletedMarker+` archive=$ARCHIVE size=$SZ" >>"$LOG"
tail -n 80 "$LOG" 2>/dev/null || true
`,
		outDir, logDir, p.MySQLPass, logPath, archive, tmpDir, mysqlDirOverride,
		p.MySQLHost, p.MySQLPort, p.MySQLUser, p.Basename,
		q(p.MySQLHost), p.MySQLPort, q(p.MySQLUser),
		q(p.MySQLHost), p.MySQLPort, q(p.MySQLUser), p.MySQLPass, p.Parallel,
		q(p.MySQLHost), p.MySQLPort, q(p.MySQLUser), p.MySQLPass, p.Parallel,
	)
}
