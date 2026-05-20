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

// shellTarGzFromDir 用 tar cf - | gzip -c 打 .tar.gz（勿 export GZIP=路径：GNU 里 GZIP 是压缩选项不是可执行文件）。
const shellTarGzFromDir = shellResolveGzip + `
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .tar.gz。请执行: yum install -y gzip" >>"$LOG"
  exit 127
}
echo "[$(date '+%%F %%T')] packing with tar|gzip (gzip=$GZIP_BIN)" >>"$LOG"
echo "[$(date '+%%F %%T')] tmp dir size: $(du -sh "$TMP" 2>/dev/null | awk '{print $1}')" >>"$LOG"
rm -f "$ARCHIVE"
_pack_exit=0
tar -cf - -C "$TMP" . 2>>"$LOG" | "$GZIP_BIN" -c > "$ARCHIVE" 2>>"$LOG" || _pack_exit=$?
if [ "$_pack_exit" -ne 0 ]; then
  echo "ERROR: tar|gzip pack failed, exit=$_pack_exit" >>"$LOG"
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
