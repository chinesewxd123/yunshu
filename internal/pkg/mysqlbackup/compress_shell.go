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

// shellTarGzFromDir 使用 tar -czf 打 gzip 包；输出追加到 $LOG（勿经 SSH stdout）。
const shellTarGzFromDir = shellResolveGzip + `
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .tar.gz。请执行: yum install -y gzip" >>"$LOG"
  exit 127
}
export GZIP="$GZIP_BIN"
echo "[$(date '+%%F %%T')] packing with tar -czf (gzip=$GZIP_BIN)" >>"$LOG"
echo "[$(date '+%%F %%T')] tmp dir size: $(du -sh "$TMP" 2>/dev/null | awk '{print $1}')" >>"$LOG"
rm -f "$ARCHIVE"
if ! tar -czf "$ARCHIVE" -C "$TMP" . >>"$LOG" 2>&1; then
  echo "ERROR: tar -czf failed, exit=$?" >>"$LOG"
  rm -f "$ARCHIVE"
  exit 1
fi
if [ ! -s "$ARCHIVE" ]; then
  echo "ERROR: 打包失败，归档为空: $ARCHIVE" >>"$LOG"
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
