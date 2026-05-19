package mysqlbackup

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var backupNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// BuildArtifactNamePrefix 备份文件名前缀（项目名_IP_端口_），用于按实例匹配已有备份。
func BuildArtifactNamePrefix(projectName, mysqlHost string, mysqlPort int) string {
	projectName = sanitizeBackupNameSegment(projectName)
	host := sanitizeBackupHost(mysqlHost)
	if mysqlPort <= 0 {
		mysqlPort = 3306
	}
	return fmt.Sprintf("%s_%s_%d_", projectName, host, mysqlPort)
}

// BuildArtifactBasename 生成备份文件基名：项目名_IP_端口_年月日_时分秒（UTC）。
func BuildArtifactBasename(projectName, mysqlHost string, mysqlPort int, at time.Time) string {
	projectName = sanitizeBackupNameSegment(projectName)
	host := sanitizeBackupHost(mysqlHost)
	if mysqlPort <= 0 {
		mysqlPort = 3306
	}
	ts := at.UTC().Format("20060102_150405")
	return fmt.Sprintf("%s_%s_%d_%s", projectName, host, mysqlPort, ts)
}

func sanitizeBackupNameSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "project"
	}
	s = backupNameUnsafe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	if s == "" {
		return "project"
	}
	return s
}

func sanitizeBackupHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "127.0.0.1"
	}
	host = strings.ReplaceAll(host, ":", "_")
	return backupNameUnsafe.ReplaceAllString(host, "_")
}
