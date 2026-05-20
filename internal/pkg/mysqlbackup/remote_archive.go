package mysqlbackup

import "fmt"

const minBackupArchiveBytes int64 = 1024

// ErrArchiveTooSmall 打包失败或压缩工具缺失时常出现 0 字节占位文件。
func ErrArchiveTooSmall(path string, size int64) error {
	return fmt.Errorf("backup archive invalid or empty (%d bytes) at %s: install pigz/gzip on backup host", size, path)
}

// ValidateArchiveSize 校验远端归档大小（SSH 执行后、WaitRemoteFile 前调用，避免空文件长时间等待）。
func ValidateArchiveSize(size int64, remotePath string) error {
	if size < minBackupArchiveBytes {
		return ErrArchiveTooSmall(remotePath, size)
	}
	return nil
}
