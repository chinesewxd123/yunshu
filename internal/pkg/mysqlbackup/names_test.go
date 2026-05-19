package mysqlbackup

import (
	"strings"
	"testing"
	"time"
)

func TestBuildArtifactBasename(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 19, 12, 33, 12, 0, time.UTC)
	base := BuildArtifactBasename("测试项目", "175.178.156.23", 3306, at)
	if !strings.HasPrefix(base, "_175.178.156.23_3306_20260519_123312") && !strings.Contains(base, "175.178.156.23_3306_20260519_123312") {
		t.Fatalf("unexpected basename: %q", base)
	}
	if strings.Contains(base, "yunshu_mysql") {
		t.Fatal("should not use legacy prefix")
	}
}
