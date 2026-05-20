package mysqlbackup

// BackupCompletedMarker 写入备份日志末行，用于与 xtrabackup 自身的 "completed OK!" 区分。
const BackupCompletedMarker = "yunshu backup completed OK!"

// shellResolveGzip 解析 gzip（兼容非交互 SSH 下 PATH 不含 /usr/bin）。
const shellResolveGzip = `
resolve_gzip() {
  if command -v gzip >/dev/null 2>&1; then command -v gzip; return 0; fi
  if [ -x /usr/bin/gzip ]; then echo /usr/bin/gzip; return 0; fi
  return 1
}
`

// shellTarGzFromDir 使用 tar -czf 直接打 gzip 包（不用 tar -I "pigz -1"，避免 Cannot exec）。
const shellTarGzFromDir = shellResolveGzip + `
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .tar.gz。请执行: yum install -y gzip"
  exit 127
}
export GZIP="$GZIP_BIN"
echo "[$(date '+%%F %%T')] packing with tar -czf (gzip=$GZIP_BIN)"
rm -f "$ARCHIVE"
tar -czf "$ARCHIVE" -C "$TMP" .
if [ ! -s "$ARCHIVE" ]; then
  echo "ERROR: 打包失败，归档为空: $ARCHIVE"
  rm -f "$ARCHIVE"
  exit 1
fi
`

// shellMysqldumpToGz 将 mysqldump 输出用 gzip 压缩为 .sql.gz。
const shellMysqldumpToGz = shellResolveGzip + `
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .sql.gz。请执行: yum install -y gzip" >>"$LOG"
  exit 127
}
mysqldump -h%s -P%d -u%s %s %s 2>>"$LOG" | "$GZIP_BIN" -1 -c > "$SQL"
`
