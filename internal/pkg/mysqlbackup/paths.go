package mysqlbackup

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateBackupPathIsolation 防止 mysqldump 落盘目录与 xtrabackup 目录重叠。
func ValidateBackupPathIsolation(mysqldumpWorkDir, remoteDataDir, remoteLogDir string) error {
	work, err := NormalizeMysqldumpWorkDir(mysqldumpWorkDir)
	if err != nil {
		return err
	}
	data := strings.TrimSpace(strings.TrimSuffix(remoteDataDir, "/"))
	logDir := strings.TrimSpace(strings.TrimSuffix(remoteLogDir, "/"))
	if data == "" && logDir == "" {
		return nil
	}
	if pathsOverlap(work, data) {
		return fmt.Errorf("mysqldump 落盘目录不能与 xtrabackup 数据目录相同或互为子目录")
	}
	if pathsOverlap(work, logDir) {
		return fmt.Errorf("mysqldump 落盘目录不能与 xtrabackup 日志目录相同或互为子目录")
	}
	return nil
}

func pathsOverlap(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	// 远端路径为 Linux 绝对路径，统一用 / 比较（避免 Windows filepath.Clean 破坏前缀判断）。
	a = normalizeUnixAbsPath(a)
	b = normalizeUnixAbsPath(b)
	if a == b {
		return true
	}
	return strings.HasPrefix(a+"/", b+"/") || strings.HasPrefix(b+"/", a+"/")
}

func normalizeUnixAbsPath(p string) string {
	p = strings.TrimSpace(filepath.ToSlash(p))
	return strings.TrimSuffix(p, "/")
}
