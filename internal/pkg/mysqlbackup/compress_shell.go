package mysqlbackup

// BackupCompletedMarker 写入备份日志末行，用于与 xtrabackup 自身的 "completed OK!" 区分。
const BackupCompletedMarker = "yunshu backup completed OK!"

// shellTarGzFromDir 将目录打包为 .tar.gz（需 pigz 或 gzip）。
const shellTarGzFromDir = `
if command -v pigz >/dev/null 2>&1; then
  tar -I "pigz -1" -cf "$ARCHIVE" -C "$TMP" .
elif command -v gzip >/dev/null 2>&1; then
  tar -I "gzip -1" -cf "$ARCHIVE" -C "$TMP" .
else
  echo "ERROR: 未找到 pigz/gzip，无法生成 .tar.gz。请安装: yum install -y gzip pigz  或  apt-get install -y gzip pigz"
  exit 127
fi
if [ ! -s "$ARCHIVE" ]; then
  echo "ERROR: 打包失败，归档为空: $ARCHIVE"
  rm -f "$ARCHIVE"
  exit 1
fi
`

// shellMysqldumpToGz 将 mysqldump 输出压缩为 .sql.gz。
const shellMysqldumpToGz = `
if command -v pigz >/dev/null 2>&1; then
  mysqldump -h%s -P%d -u%s %s %s 2>>"$LOG" | pigz -1 -c > "$SQL"
elif command -v gzip >/dev/null 2>&1; then
  mysqldump -h%s -P%d -u%s %s %s 2>>"$LOG" | gzip -1 -c > "$SQL"
else
  echo "ERROR: 未找到 pigz/gzip，无法生成 .sql.gz。请安装: yum install -y gzip pigz  或  apt-get install -y gzip pigz" >>"$LOG"
  exit 127
fi
`
