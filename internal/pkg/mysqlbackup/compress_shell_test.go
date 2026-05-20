package mysqlbackup

import (
	"strings"
	"testing"
)

func TestXtrabackupScriptRequiresGzip(t *testing.T) {
	script := BuildXtrabackupRemoteScript(XtrabackupRemoteScriptParams{
		DataDir: "/data", LogDir: "/log", Basename: "test",
		MySQLHost: "127.0.0.1", MySQLPort: 3306, MySQLUser: "root", MySQLPass: "'x'",
		ShellQuote: func(s string) string { return "'" + s + "'" },
	})
	for _, sub := range []string{
		"command -v gzip",
		"未找到 pigz/gzip",
		BackupCompletedMarker,
		`[ ! -s "$ARCHIVE" ]`,
	} {
		if !strings.Contains(script, sub) {
			t.Fatalf("script missing %q", sub)
		}
	}
}
